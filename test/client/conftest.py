import os
import shutil
from os.path import abspath, dirname, isdir, join
from subprocess import call

import pytest

from app import App
from files import read
from rustdesk import LOG_DIR, RustDesk
from screen import Screen

ROOT = abspath(join(dirname(__file__), '..', '..'))
STATE_DIR = join(ROOT, 'web', 'e2e', '.state')


def pytest_addoption(parser):
    parser.addoption('--domain', action='store')
    parser.addoption('--role', action='store')


@pytest.fixture(scope='session')
def role(request):
    return request.config.getoption('--role')


@pytest.fixture(scope='session')
def artifact_dir():
    path = join(ROOT, 'artifact', 'client')
    os.makedirs(path, exist_ok=True)
    return path


@pytest.fixture(scope='session')
def password():
    return 'Remote-Password1'


@pytest.fixture(scope='session')
def workstation_id_file(artifact_dir):
    return join(artifact_dir, 'workstation-id.txt')


@pytest.fixture(scope='session')
def workstation_done_file(artifact_dir):
    return join(artifact_dir, 'workstation-done')


@pytest.fixture(scope='session')
def server_config():
    return read(join(STATE_DIR, 'config.txt'))


@pytest.fixture(scope='session')
def token():
    return read(join(STATE_DIR, 'token.txt'))


@pytest.fixture(scope='session')
def app(request, token):
    return App(request.config.getoption('--domain'), token)


@pytest.fixture(scope='session')
def screen(role, artifact_dir):
    screenshot_dir = join(ROOT, 'artifact', 'screenshots')
    screen = Screen(screenshot_dir)
    screen.wait()
    yield screen
    call(['import', '-window', 'root', screen.path('{0}-exit'.format(role))])
    call(['chmod', '-R', 'a+r', artifact_dir, screenshot_dir])


@pytest.fixture(scope='session')
def rustdesk(role, artifact_dir, screen):
    yield RustDesk()
    if isdir(LOG_DIR):
        shutil.copytree(LOG_DIR, join(artifact_dir, '{0}-logs'.format(role)), dirs_exist_ok=True)
