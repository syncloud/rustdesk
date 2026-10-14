import pytest
import requests
from subprocess import check_output
from retrying import retry
from syncloudlib.integration.hosts import add_host_alias
from test import snapd
from test import websocket

TMP_DIR = '/tmp/syncloud'


@pytest.fixture(scope="session")
def module_setup(request, device, artifact_dir):
    def module_teardown():
        device.run_ssh('mkdir -p {0}'.format(TMP_DIR), throw=False)
        device.run_ssh('journalctl > {0}/upgrade-prev.journalctl.log'.format(TMP_DIR), throw=False)
        device.run_ssh('snap logs rustdesk -n 500 > {0}/upgrade-prev.snap.log'.format(TMP_DIR), throw=False)
        device.scp_from_device('{0}/*'.format(TMP_DIR), artifact_dir)
        check_output('chmod -R a+r {0}'.format(artifact_dir), shell=True)

    request.addfinalizer(module_teardown)


@retry(stop_max_attempt_number=30, wait_fixed=2000)
def index(app_domain):
    response = requests.get('https://{0}'.format(app_domain), verify=False, allow_redirects=False)
    assert response.status_code == 200, response.text


def test_start(module_setup, device, app, domain, device_host):
    add_host_alias(app, device_host, domain)
    device.activated()
    snapd.settle(device)


def test_install_from_store(device):
    device.run_ssh('snap remove --purge rustdesk')
    device.run_ssh('snap install rustdesk', retries=10)


def test_installed_revision_is_from_store(device):
    revision = device.run_ssh('snap list rustdesk').strip().splitlines()[-1].split()[2]
    assert revision.isdigit(), 'expected a store revision, got {0}'.format(revision)


@pytest.mark.parametrize('path', ['/ws/id', '/ws/relay'])
def test_server_running(app_domain, path):
    websocket.upgrade(app_domain, path)


def test_index(app_domain):
    index(app_domain)
