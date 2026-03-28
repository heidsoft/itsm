/**
 * Alertmanager Webhook Receiver
 * Receives alerts from Alertmanager and forwards to ITSM backend
 */

const http = require('http');
const https = require('https');

const ITSM_BACKEND_URL = process.env.ITSM_BACKEND_URL || 'http://itsm-backend:8080';
const PORT = process.env.PORT || 9094;

// Health check mode
if (process.argv.includes('--health')) {
  console.log('Health check passed');
  process.exit(0);
}

function parseAlertmanagerPayload(alertmanagerData) {
  const alerts = alertmanagerData.alerts || [];

  return alerts.map(alert => ({
    alertName: alert.labels?.alertname || 'Unknown',
    status: alert.status || 'unknown',
    severity: alert.labels?.severity || 'info',
    service: alert.labels?.service || alert.labels?.job || 'unknown',
    cluster: alert.labels?.cluster || 'unknown',
    namespace: alert.labels?.namespace || 'default',
    description: alert.annotations?.description || alert.annotations?.summary || '',
    startsAt: alert.startsAt || new Date().toISOString(),
    endsAt: alert.endsAt || null,
    generatorURL: alert.generatorURL || '',
    labels: alert.labels || {},
  }));
}

function forwardToITSM(alertData) {
  const data = JSON.stringify({
    source: 'alertmanager',
    event_type: 'alert',
    data: alertData,
    timestamp: new Date().toISOString(),
  });

  const url = new URL(`${ITSM_BACKEND_URL}/api/v1/alerts/webhook`);
  const protocol = url.protocol === 'https:' ? https : http;

  const options = {
    hostname: url.hostname,
    port: url.port || (url.protocol === 'https:' ? 443 : 80),
    path: url.pathname,
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Content-Length': Buffer.byteLength(data),
      'X-Forwarded-By': 'alert-webhook-receiver',
    },
    timeout: 10000,
  };

  return new Promise((resolve, reject) => {
    const req = protocol.request(options, (res) => {
      let body = '';
      res.on('data', chunk => body += chunk);
      res.on('end', () => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve({ success: true, statusCode: res.statusCode, body });
        } else {
          reject(new Error(`HTTP ${res.statusCode}: ${body}`));
        }
      });
    });

    req.on('error', reject);
    req.on('timeout', () => {
      req.destroy();
      reject(new Error('Request timeout'));
    });

    req.write(data);
    req.end();
  });
}

const server = http.createServer(async (req, res) => {
  // Health check
  if (req.url === '/health' && req.method === 'GET') {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end('ok');
    return;
  }

  // CORS preflight
  if (req.method === 'OPTIONS') {
    res.writeHead(204, {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type',
      'Access-Control-Max-Age': '86400',
    });
    res.end();
    return;
  }

  // Only accept POST to /webhook
  if (req.url !== '/webhook' || req.method !== 'POST') {
    res.writeHead(404, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'Not found' }));
    return;
  }

  let body = '';

  req.on('data', chunk => {
    body += chunk.toString();
  });

  req.on('end', async () => {
    const corsHeaders = {
      'Access-Control-Allow-Origin': '*',
      'Content-Type': 'application/json',
    };

    try {
      const alertmanagerData = JSON.parse(body);
      const itsmAlerts = parseAlertmanagerPayload(alertmanagerData);

      console.log(`Received ${itsmAlerts.length} alerts from Alertmanager`);

      // Forward each alert to ITSM backend
      const results = await Promise.allSettled(
        itsmAlerts.map(alert => forwardToITSM(alert))
      );

      const successful = results.filter(r => r.status === 'fulfilled').length;
      const failed = results.filter(r => r.status === 'rejected').length;

      console.log(`Forwarded: ${successful} successful, ${failed} failed`);

      res.writeHead(200, { ...corsHeaders });
      res.end(JSON.stringify({
        status: 'success',
        received: itsmAlerts.length,
        forwarded: successful,
        failed,
      }));
    } catch (error) {
      console.error('Error processing alert:', error.message);

      res.writeHead(500, { ...corsHeaders });
      res.end(JSON.stringify({
        status: 'error',
        message: error.message,
      }));
    }
  });
});

server.listen(PORT, () => {
  console.log(`Alert Webhook Receiver listening on port ${PORT}`);
  console.log(`Forwarding alerts to ${ITSM_BACKEND_URL}`);
});

server.on('error', (error) => {
  console.error('Server error:', error);
  process.exit(1);
});
