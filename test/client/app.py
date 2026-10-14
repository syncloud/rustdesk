import ssl
from urllib.request import Request, urlopen


class App:

    def __init__(self, domain, token):
        self.domain = domain
        self.token = token

    def address_book(self):
        request = Request('https://{0}/api/ab'.format(self.domain),
                          headers={'Authorization': 'Bearer {0}'.format(self.token)})
        context = ssl.create_default_context()
        context.check_hostname = False
        context.verify_mode = ssl.CERT_NONE
        return urlopen(request, context=context, timeout=30).read().decode()
