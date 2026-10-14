import os
import pytest
import requests
from os.path import join
from subprocess import check_output
from retrying import retry
from test import snapd
from test import websocket
from syncloudlib.integration.hosts import add_host_alias
from syncloudlib.integration.installer import local_install

TMP_DIR = '/tmp/syncloud'

DATA_DIR = '/var/snap/rustdesk/current'
KEY_FILE = '{0}/server/id_ed25519.pub'.format(DATA_DIR)


@pytest.fixture(scope="session")
def module_setup(request, device, platform_data_dir, artifact_dir):
    def module_teardown():
        platform_log_dir = join(artifact_dir, 'platform_log')
        os.mkdir(platform_log_dir)
        device.scp_from_device('{0}/log/*'.format(platform_data_dir), platform_log_dir, throw=False)
        device.run_ssh('mkdir -p {0}'.format(TMP_DIR), throw=False)
        device.run_ssh('journalctl > {0}/journalctl.log'.format(TMP_DIR), throw=False)
        device.run_ssh('snap logs rustdesk -n 500 > {0}/snap.rustdesk.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ps auxfw > {0}/ps.log'.format(TMP_DIR), throw=False)
        device.run_ssh('netstat -nlp > {0}/netstat.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la {0} > {1}/data.ls.log'.format(DATA_DIR, TMP_DIR), throw=False)
        device.run_ssh('ls -la {0}/server > {1}/server.ls.log'.format(DATA_DIR, TMP_DIR), throw=False)
        device.run_ssh('ls -la {0}/config > {1}/config.ls.log'.format(DATA_DIR, TMP_DIR), throw=False)
        device.scp_from_device('{0}/*'.format(TMP_DIR), artifact_dir)
        check_output('chmod -R a+r {0}'.format(artifact_dir), shell=True)

    request.addfinalizer(module_teardown)


@pytest.fixture(scope="session")
def state():
    return {}


def key(device):
    return device.run_ssh('cat {0}'.format(KEY_FILE)).strip()


def test_start(module_setup, device, app, domain, device_host):
    add_host_alias(app, device_host, domain)
    device.run_ssh('date', retries=100, throw=True)
    snapd.settle(device)


@retry(stop_max_attempt_number=30, wait_fixed=10000)
def activate(device):
    response = device.activate_custom()
    assert response.status_code == 200, response.text


def test_activate_device(device):
    activate(device)


def test_install(app_archive_path, device_host, device_session, device_password):
    local_install(device_host, device_password, app_archive_path)


def test_index(app_domain):
    response = requests.get('https://{0}'.format(app_domain), verify=False, allow_redirects=False)
    assert response.status_code == 200, response.text


def test_backend_ready(device):
    device.run_ssh('test -S {0}/backend.sock'.format(DATA_DIR), retries=100)


def test_key_generated(device, state):
    device.run_ssh('test -f {0}'.format(KEY_FILE), retries=100)
    state['key'] = key(device)
    assert len(state['key']) == 44, state['key']


def test_private_key_is_not_world_readable(device):
    mode = device.run_ssh('stat -c %a:%U {0}/server/id_ed25519'.format(DATA_DIR)).strip()
    assert mode == '600:rustdesk', mode


def test_backend_env_is_not_world_readable(device):
    mode = device.run_ssh('stat -c %a:%U {0}/config/backend.env'.format(DATA_DIR)).strip()
    assert mode == '600:rustdesk', mode


@pytest.mark.parametrize('name', ['hbbs', 'hbbr'])
def test_daemon_listens_on_its_socket(device, name):
    device.run_ssh('test -S {0}/{1}.sock'.format(DATA_DIR, name), retries=100)
    mode = device.run_ssh('stat -c %a:%U {0}/{1}.sock'.format(DATA_DIR, name)).strip()
    assert mode == '700:rustdesk', mode


def test_no_network_port_is_opened(device):
    listeners = device.run_ssh('netstat -nltup')
    assert 'hbb' not in listeners, listeners


@pytest.mark.parametrize('path', ['/ws/id', '/ws/relay'])
def test_websocket_reachable_over_https(app_domain, path):
    websocket.upgrade(app_domain, path)


def test_services_run_as_app_user(device):
    owners = {}
    for line in device.run_ssh('ps -eo user:20,args').splitlines():
        fields = line.split()
        if len(fields) > 1 and fields[1].startswith('/snap/rustdesk/'):
            owners[fields[1].split('/')[4]] = fields[0]
    assert owners == {'rustdesk': 'rustdesk', 'backend': 'rustdesk', 'nginx': 'rustdesk'}, owners


def test_api_requires_login(app_domain):
    response = requests.get('https://{0}/api/server'.format(app_domain), verify=False)
    assert response.status_code == 401, response.text


def test_client_api_offers_syncloud_login(app_domain):
    response = requests.get('https://{0}/api/login-options'.format(app_domain), verify=False)
    assert response.status_code == 200, response.text
    assert response.json() == ['oidc/syncloud']


def test_client_login_stays_pending_without_approval(app_domain):
    device = {'id': '900800700', 'uuid': 'cHl0ZXN0'}
    started = requests.post('https://{0}/api/oidc/auth'.format(app_domain), verify=False,
                            json=dict(device, op='syncloud', deviceInfo={'os': 'linux', 'name': 'pytest'}))
    assert started.status_code == 200, started.text
    assert started.json()['url'].startswith('https://{0}/device?code='.format(app_domain))
    query = requests.get('https://{0}/api/oidc/auth-query'.format(app_domain), verify=False,
                         params=dict(device, code=started.json()['code']))
    assert query.json() == {'error': 'No authed oidc is found'}, query.text


def test_client_login_approval_requires_web_session(app_domain):
    response = requests.post('https://{0}/api/device-login'.format(app_domain), verify=False,
                             json={'code': 'anything'})
    assert response.status_code == 401, response.text


def test_address_book_requires_token(app_domain):
    response = requests.get('https://{0}/api/ab'.format(app_domain), verify=False,
                            headers={'Authorization': 'Bearer forged'})
    assert response.status_code == 401, response.text


def test_unknown_client_api_is_404(app_domain):
    response = requests.post('https://{0}/api/ab/personal'.format(app_domain), verify=False)
    assert response.status_code == 404, response.text


def test_account_data_is_not_world_readable(device, app_domain):
    response = requests.post('https://{0}/api/sysinfo'.format(app_domain), verify=False,
                             json={'id': '900800700', 'uuid': 'cHl0ZXN0', 'hostname': 'pytest'})
    assert response.text == 'SYSINFO_UPDATED', response.text
    mode = device.run_ssh('stat -c %a:%U {0}/account'.format(DATA_DIR)).strip()
    assert mode == '700:rustdesk', mode
    mode = device.run_ssh('stat -c %a:%U {0}/account/account.db'.format(DATA_DIR)).strip()
    assert mode == '600:rustdesk', mode


def test_auth_login_redirects_to_authelia(app_domain):
    response = requests.get('https://{0}/auth/login'.format(app_domain),
                            verify=False, allow_redirects=False)
    assert response.status_code == 302, \
        'expected a redirect to Authelia; 503 means OIDC discovery never succeeded ' \
        'and sign-in is broken even though the backend is serving: {0}'.format(response.text)
    assert 'auth.' in response.headers.get('Location', ''), response.headers


def test_upgrade_same_version(app_archive_path, device_host, device_password):
    local_install(device_host, device_password, app_archive_path)


def test_key_survives_refresh(device, state):
    assert key(device) == state['key'], \
        'the key changed on refresh, every configured client is now locked out'


def test_remove(device, app):
    response = device.app_remove(app)
    assert response.status_code == 200, response.text


def test_reinstall(app_archive_path, device_host, device_password):
    local_install(device_host, device_password, app_archive_path)


def test_key_generated_after_reinstall(device):
    device.run_ssh('test -f {0}'.format(KEY_FILE), retries=100)
