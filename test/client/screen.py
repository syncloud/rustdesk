import logging
import os
from os.path import join
from subprocess import DEVNULL, call, check_call

from wait import wait_for

log = logging.getLogger(__name__)


class Screen:

    def __init__(self, screenshot_dir):
        self.screenshot_dir = screenshot_dir
        os.makedirs(screenshot_dir, exist_ok=True)

    def wait(self):
        wait_for('the display', lambda: call(['xdpyinfo'], stdout=DEVNULL) == 0)

    def path(self, name):
        return join(self.screenshot_dir, '{0}.png'.format(name))

    def shoot(self, name):
        log.info('screenshot %s', name)
        temp = join(self.screenshot_dir, '{0}.tmp.png'.format(name))
        check_call(['import', '-window', 'root', temp])
        os.rename(temp, self.path(name))
