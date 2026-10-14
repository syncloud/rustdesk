local name = 'rustdesk';

local go = '1.26';
local node = '20.18.1';
local nginx = '1.29.3-alpine3.22';
local debian = 'bookworm-slim';
local python = '3.12-slim-bookworm';
local playwright = 'mcr.microsoft.com/playwright:v1.59.1-jammy';
local store_publisher = 'stable-346';

local rustdesk = '1.1.17-syncloud.1';
local rustdesk_client = '1.5.0';
local ubuntu = '24.04';

local platform = '26.08.01';
local distro_default = 'bookworm';
local distros = ['bookworm', 'buster'];

local platform_image(distro) = 'syncloud/platform-' + distro + ':' + platform;

local e2e(step, project, spec, prefix='') = {
  name: step,
  image: playwright,
  commands: ['./ci/ui.sh ' + project + ' ' + name + ' ' + distro_default + ' ' + spec],
  environment: { DEVICE_USER: 'user', DEVICE_PASSWORD: 'Password1', SCREENSHOT_PREFIX: prefix },
};

local build(arch, test_ui) = [{
  kind: 'pipeline',
  type: 'docker',
  name: arch,
  platform: {
    os: 'linux',
    arch: arch,
  },
  steps: [
    {
      name: 'nginx',
      image: 'nginx:' + nginx,
      commands: ['./nginx/build.sh'],
    },
  ] + [
    {
      name: 'nginx test ' + distro,
      image: platform_image(distro),
      commands: ['./nginx/test.sh'],
    }
    for distro in distros
  ] + [
    {
      name: 'rustdesk',
      image: 'debian:' + debian,
      commands: ['./rustdesk/build.sh ' + rustdesk],
    },
  ] + [
    {
      name: 'rustdesk test ' + distro,
      image: platform_image(distro),
      commands: ['./rustdesk/test.sh'],
    }
    for distro in distros
  ] + [
    {
      name: 'cli',
      image: 'golang:' + go,
      commands: ['./cli/build.sh'],
    },
  ] + [
    {
      name: 'cli test ' + distro,
      image: platform_image(distro),
      commands: ['./cli/test.sh'],
    }
    for distro in distros
  ] + [
    {
      name: 'backend',
      image: 'golang:' + go,
      commands: ['./backend/build.sh'],
    },
    {
      name: 'web',
      image: 'node:' + node,
      commands: ['./web/build.sh'],
    },
    {
      name: 'package',
      image: 'debian:' + debian,
      commands: ['./package.sh ' + name + ' $DRONE_BUILD_NUMBER'],
    },
  ] + [
    {
      name: 'test ' + distro,
      image: 'python:' + python,
      commands: ['./ci/test.sh test.py ' + distro + ' ' + name],
    }
    for distro in distros
  ] + (if test_ui then [
         e2e('e2e', 'desktop', 'specs/01-smoke.spec.ts'),
         {
           name: 'test-upgrade-prev',
           image: 'python:' + python,
           commands: ['./ci/test.sh upgrade_prev.py ' + distro_default + ' ' + name],
         },
         e2e('e2e-prev', 'desktop', 'specs/01-smoke.spec.ts', 'store-'),
         {
           name: 'workstation',
           image: 'ubuntu:' + ubuntu,
           detach: true,
           commands: ['./ci/client.sh workstation ' + rustdesk_client + ' ' + name + '.' + distro_default + '.com'],
         },
         {
           name: 'client',
           image: 'ubuntu:' + ubuntu,
           commands: ['./ci/client.sh control ' + rustdesk_client + ' ' + name + '.' + distro_default + '.com'],
         },
         e2e('e2e-before-upgrade', 'desktop', 'specs/02-pre-upgrade.spec.ts'),
         {
           name: 'test-upgrade',
           image: 'python:' + python,
           commands: ['./ci/test.sh upgrade.py ' + distro_default + ' ' + name],
         },
         e2e('e2e-after-upgrade', 'desktop', 'specs/03-post-upgrade.spec.ts'),
       ] else []) + [
    {
      name: 'publish',
      image: 'syncloud/store-publisher:' + store_publisher,
      environment: {
        SYNCLOUD_TOKEN: { from_secret: 'SYNCLOUD_TOKEN' },
      },
      command: ['snap', '-c', '${DRONE_BRANCH}'],
      when: {
        branch: ['stable'],
        event: ['push'],
      },
    },
    {
      name: 'artifact',
      image: 'appleboy/drone-scp:1.6.4',
      settings: {
        host: { from_secret: 'artifact_host' },
        username: 'artifact',
        key: { from_secret: 'artifact_key' },
        timeout: '2m',
        command_timeout: '2m',
        target: '/home/artifact/repo/' + name + '/${DRONE_BUILD_NUMBER}-' + arch,
        source: ['artifact/*'],
        strip_components: 1,
      },
      when: {
        status: ['failure', 'success'],
      },
    },
  ],
  trigger: {
    event: ['push'],
  },
  services: [
    {
      name: name + '.' + distro + '.com',
      image: platform_image(distro),
      privileged: true,
      entrypoint: ['/bin/sh', '-c', "mkdir -p /etc/systemd/system/snapd.service.d && printf '[Service]\\nExecStartPost=/bin/sh -c \"/usr/bin/snap set system refresh.hold=2099-01-01T00:00:00Z\"\\n' > /etc/systemd/system/snapd.service.d/disable-refresh.conf && exec /sbin/init"],
      volumes: [
        { name: 'dbus', path: '/var/run/dbus' },
        { name: 'dev', path: '/dev' },
      ],
    }
    for distro in distros
  ],
  volumes: [
    { name: 'dbus', host: { path: '/var/run/dbus' } },
    { name: 'dev', host: { path: '/dev' } },
  ],
}];

build('amd64', true) +
build('arm64', false) +
build('arm', false)
