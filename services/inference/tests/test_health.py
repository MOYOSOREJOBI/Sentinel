import unittest

from app import app


class HealthTest(unittest.TestCase):
    def setUp(self):
        self.client = app.test_client()

    def test_healthz_alive(self):
        resp = self.client.get('/healthz')
        self.assertEqual(resp.status_code, 200)
        self.assertIn('ok', resp.get_data(as_text=True))


if __name__ == '__main__':
    unittest.main()
