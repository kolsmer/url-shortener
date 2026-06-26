import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  // stages: [
  //   { duration: '30s', target: 200 },   // Ramp up to 200 users
  //   { duration: '1m30s', target: 200 }, // Stay at 200 users
  //   { duration: '30s', target: 400 },   // Ramp up to 400 users
  //   { duration: '1m30s', target: 400 }, // Stay at 400 users
  //   { duration: '30s', target: 0 },     // Ramp down to 0
  // ],
   stages: [
    { duration: '30s', target: 200 },   // ramp up
    { duration: '1m', target: 200 },     // steady 200
    { duration: '20s', target: 0 },      // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000', 'p(99)<2000'],
    http_req_failed: ['rate<0.2'],
  },
};

const BASE_URL = __ENV.TARGET_URL || 'http://localhost:8080';

export function setup() {
  http.post(`${BASE_URL}/api/v1/register`, JSON.stringify({
    email: 'loadtest@example.com',
    password: 'secret123',
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  const res = http.post(`${BASE_URL}/api/v1/login`, JSON.stringify({
    email: 'loadtest@example.com',
    password: 'secret123',
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  return { token: JSON.parse(res.body).token };
}


export default function (data) {
  // Test shorten endpoint
  const shortenRes = http.post(`${BASE_URL}/api/v1/shorten`, JSON.stringify({
    url: `https://example${__VU}.com/${__ITER}`,
  }), {
    headers: { 'Content-Type': 'application/json',
      'Authorization': `Bearer ${data.token}`
    },
  });

  check(shortenRes, {
    'POST shorten status 201': (r) => r.status === 201,
    'POST shorten has code': (r) => r.body.includes('"code"'),
  });

  let code;
  try {
    code = JSON.parse(shortenRes.body).code;
  }
  catch (e) {
    return;
  }

  // Minimal sleep
  sleep(0.05);

  // Test redirect endpoint
  const getRes = http.get(`${BASE_URL}/${code}`, {
    redirects: 0,
  });

  check(getRes, {
    'GET redirect status 307': (r) => r.status === 307,
    'GET has location': (r) => r.headers.Location !== undefined,
  });

  sleep(0.05);
}
