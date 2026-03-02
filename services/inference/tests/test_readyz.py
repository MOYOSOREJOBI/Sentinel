import unittest
from unittest.mock import patch

import app as inference_app


class ReadyzTest(unittest.TestCase):
    def setUp(self):
        self.client = inference_app.app.test_client()

    def test_readyz_fails_when_dependencies_not_ready(self):
        with patch.object(inference_app, 'model_loaded', False), patch.object(inference_app, 'kafka_ready', False):
            resp = self.client.get('/readyz')
            self.assertEqual(resp.status_code, 503)

    def test_readyz_succeeds_when_model_and_kafka_ready(self):
        with patch.object(inference_app, 'model_loaded', True), patch.object(inference_app, 'kafka_ready', True):
            resp = self.client.get('/readyz')
            self.assertEqual(resp.status_code, 200)


if __name__ == '__main__':
    unittest.main()
