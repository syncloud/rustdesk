import pytest
import requests
from subprocess import check_output
from syncloudlib.integration.hosts import add_host_alias
from syncloudlib.integration.installer import local_install
from test import snapd
from test import websocket

TMP_DIR = '/tmp/syncloud'

DATA_DIR = '/var/snap/rustdesk/current'
KEY_FILE = '{0}/server/id_ed25519.pub'.format(DATA_DIR)


@pytest.fixture(scope="session")
def module_setup(request, device, artifact_dir):
    def module_teardown():
        device.run_ssh('mkdir -p {0}'.format(TMP_DIR), throw=False)
        device.run_ssh('journalctl > {0}/upgrade.journalctl.log'.format(TMP_DIR), throw=False)
        device.run_ssh('snap logs rustdesk -n 500 > {0}/upgrade.snap.log'.format(TMP_DIR), throw=False)
        device.scp_from_device('{0}/*'.format(TMP_DIR), artifact_dir)
        check_output('chmod -R a+r {0}'.format(artifact_dir), shell=True)

    request.addfinalizer(module_teardown)


@pytest.fixture(scope="session")
def state():
    return {}


def sha256(device, file):
    return device.run_ssh("sha256sum {0} | cut -d' ' -f1".format(file)).strip()


def test_start(module_setup, device, app, domain, device_host):
    add_host_alias(app, device_host, domain)
    device.activated()
    snapd.settle(device)


def test_capture_state(device, state):
    state['public'] = sha256(device, KEY_FILE)
    state['private'] = sha256(device, '{0}/server/id_ed25519'.format(DATA_DIR))
    state['secret'] = sha256(device, '{0}/.secret'.format(DATA_DIR))
    assert state['public']
    assert state['private']
    assert state['secret']


def test_upgrade(app_archive_path, device_host, device_password):
    local_install(device_host, device_password, app_archive_path)


def test_key_is_not_regenerated(device, state):
    assert sha256(device, KEY_FILE) == state['public'], \
        'the key changed on upgrade, every configured client is now locked out'
    assert sha256(device, '{0}/server/id_ed25519'.format(DATA_DIR)) == state['private']


def test_sessions_survive_upgrade(device, state):
    assert sha256(device, '{0}/.secret'.format(DATA_DIR)) == state['secret'], \
        'the session secret changed on upgrade, every signed-in user is logged out'


@pytest.mark.parametrize('path', ['/ws/id', '/ws/relay'])
def test_server_running_after_upgrade(app_domain, path):
    websocket.upgrade(app_domain, path)


def test_auth_still_works_after_upgrade(app_domain):
    response = requests.get('https://{0}/auth/login'.format(app_domain),
                            verify=False, allow_redirects=False)
    assert response.status_code == 302, \
        'sign-in broke after the refresh; 503 means OIDC discovery failed: {0}'.format(response.text)


def test_index_after_upgrade(app_domain):
    response = requests.get('https://{0}'.format(app_domain), verify=False, allow_redirects=False)
    assert response.status_code == 200, response.text
