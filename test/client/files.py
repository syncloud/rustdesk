import os


def read(path):
    with open(path) as f:
        return f.read().strip()


def write(path, content):
    temp = '{0}.tmp'.format(path)
    with open(temp, 'w') as f:
        f.write(content)
    os.rename(temp, path)
