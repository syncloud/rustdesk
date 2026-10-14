import logging
import os
from os.path import isfile, join
from subprocess import Popen, call, check_call, check_output, run

from wait import wait_for

log = logging.getLogger(__name__)

CONFIG_DIR = '/root/.config/rustdesk'
LOCAL_CONFIG = join(CONFIG_DIR, 'RustDesk_local.toml')
LOG_DIR = '/root/.local/share/logs/RustDesk'
SERVICE_LOG = join(LOG_DIR, 'server', 'rustdesk_rCURRENT.log')


class RustDesk:

    def __init__(self):
        self.background = []

    def start(self, *args):
        log.info('start rustdesk %s', ' '.join(args))
        self.background.append(Popen(['rustdesk'] + list(args)))

    def start_service(self):
        self.start('--server')
        wait_for('the client service', lambda: call(['rustdesk', '--get-id']) == 0)

    def register(self, config):
        self.start_service()
        self.import_config(config)
        self.option('allow-websocket', 'Y')
        self.option('allow-insecure-tls-fallback', 'Y')
        wait_for('registration over websocket', lambda: self.logged('keep_alive', SERVICE_LOG))

    def id(self):
        return check_output(['rustdesk', '--get-id']).decode().strip().splitlines()[-1]

    def import_config(self, config):
        log.info('import the server config')
        check_call(['rustdesk', '--config', config])

    def option(self, name, value):
        log.info('option %s', name)
        check_call(['rustdesk', '--option', name, value])

    def local_option(self, name, value):
        log.info('local option %s', name)
        os.makedirs(CONFIG_DIR, exist_ok=True)
        lines = []
        if isfile(LOCAL_CONFIG):
            with open(LOCAL_CONFIG) as f:
                lines = f.read().splitlines()
        if '[options]' not in lines:
            lines += ['', '[options]']
        lines.insert(lines.index('[options]') + 1, "{0} = '{1}'".format(name, value))
        with open(LOCAL_CONFIG, 'w') as f:
            f.write('\n'.join(lines) + '\n')

    def set_password(self, password):
        log.info('set the permanent password')
        check_call(['rustdesk', '--password', password])

    def logged(self, text, path=LOG_DIR):
        return run(['grep', '-rqs', text, path]).returncode == 0

    def open_main_window(self):
        self.start()

    def main_windows(self):
        processes = check_output(['ps', '-eo', 'pid,args']).decode().splitlines()
        return [line.split()[0] for line in processes if line.split()[1:] == ['rustdesk']]

    def close_main_window(self):
        log.info('close the main window')
        check_call(['kill'] + self.main_windows())
        wait_for('the main window to close', lambda: not self.main_windows())

    def connect(self, peer, password):
        log.info('connect to %s', peer)
        self.start('--connect', peer, '--password', password)
        wait_for('the remote screen', lambda: self.logged('new video handler'))

    def disconnect(self):
        log.info('close the remote session')
        call(['pkill', '-f', 'rustdesk --connect'])
