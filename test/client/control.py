import time
from os.path import isfile
from subprocess import Popen

from files import read
from wait import wait_for


def test_desktop_starts(screen):
    Popen(['openbox'])


def test_registers_over_websocket(rustdesk, server_config):
    rustdesk.register(server_config)


def test_signs_in_with_the_server_token(rustdesk, screen, token):
    rustdesk.local_option('access_token', token)
    rustdesk.local_option('sync-ab-with-recent-sessions', 'Y')
    rustdesk.open_main_window()
    time.sleep(10)
    screen.shoot('control-ready')


def test_controls_the_workstation(rustdesk, screen, password, workstation_id_file, workstation_done_file):
    wait_for('the workstation', lambda: isfile(workstation_id_file), attempts=300)
    rustdesk.connect(read(workstation_id_file), password)
    time.sleep(5)
    screen.shoot('session')
    wait_for('the workstation screenshot', lambda: isfile(workstation_done_file))


def test_saves_the_workstation_to_the_address_book(rustdesk, screen, app, workstation_id_file):
    workstation = read(workstation_id_file)
    rustdesk.disconnect()
    rustdesk.close_main_window()
    rustdesk.open_main_window()
    wait_for('the workstation in the synced address book', lambda: workstation in app.address_book())
    time.sleep(5)
    screen.shoot('control-signed-in')
