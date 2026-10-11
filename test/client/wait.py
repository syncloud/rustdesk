import time


def wait_for(what, check, attempts=60, delay=2):
    for _ in range(attempts):
        if check():
            return
        time.sleep(delay)
    raise AssertionError('timed out waiting for {0}'.format(what))
