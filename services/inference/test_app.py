import unittest

from app import app


class InferenceAppTest(unittest.TestCase):
    def setUp(self):
        self.client = app.test_client()

    def test_metrics_endpoint_exists(self):
        resp = self.client.get('/metrics')
        self.assertEqual(resp.status_code, 200)
        self.assertIn('inference_messages_consumed_total', resp.get_data(as_text=True))


if __name__ == '__main__':
    unittest.main()
