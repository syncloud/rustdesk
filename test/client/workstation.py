import time
from os.path import isfile
from subprocess import DEVNULL, Popen, call

from files import write
from wait import wait_for


def test_desktop_starts(screen):
    Popen(['dbus-launch', '--exit-with-session', 'startxfce4'])
    wait_for('the desktop', lambda: call(['pgrep', 'xfce4-panel'], stdout=DEVNULL) == 0)


def test_registers_over_websocket(rustdesk, server_config):
    rustdesk.register(server_config)


def test_main_window_opens(rustdesk, screen):
    rustdesk.open_main_window()
    time.sleep(10)
    screen.shoot('workstation-ready')


def test_publishes_its_id_and_password(rustdesk, password, workstation_id_file):
    rustdesk.set_password(password)
    Popen(['xfce4-terminal', '--geometry', '70x8+40+60', '--title', 'workstation', '-e',
           "sh -c 'echo this workstation is controlled over websocket; hostname; sleep infinity'"])
    write(workstation_id_file, rustdesk.id())


def test_is_controlled(screen, workstation_done_file):
    wait_for('the controlling client to finish', lambda: isfile(screen.path('session')), attempts=300)
    screen.shoot('workstation-controlled')
    write(workstation_done_file, '')
