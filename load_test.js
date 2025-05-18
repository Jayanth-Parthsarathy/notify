import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '10s', target: 20 },  // Ramp up to 20 users over 10 seconds
    { duration: '20s', target: 20 },  // Hold at 20 users for 20 seconds
    { duration: '10s', target: 0 },   // Ramp down
  ],
};

export default function () {
  const url = 'http://localhost:8090/notify';
  const payload = JSON.stringify({
    email: `user${__VU}@example.com`,
    message: 'Hello from K6 load test!'
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  let res = http.post(url, payload, params);

  check(res, {
    'status is 200': (r) => r.status === 200,
  });

  // sleep(1); // Wait 1s between requests per VU
}
