import { check, sleep } from 'k6';
import http from 'k6/http';

export const options = {
  // A simple baseline test: 10 concurrent virtual users for 30 seconds
  vus: 10,
  duration: '30s',
};

export default function () {
  const baseUrl = 'http://localhost:3000';
  
  // Check the health endpoint
  const healthRes = http.get(`${baseUrl}/health`);
  check(healthRes, {
    'health status is 200': (r) => r.status === 200,
  });

  // Check GetUser endpoint via gRPC-gateway
  const userRes = http.get(`${baseUrl}/v1/users/123`);
  check(userRes, {
    'getUser status is 200': (r) => r.status === 200,
    'getUser response contains ID': (r) => r.json('id') === '123',
    'getUser response contains Name': (r) => r.json('name') === 'User-123',
  });

  // Check a non-existent route
  const notFoundRes = http.get(`${baseUrl}/this-route-does-not-exist`);
  check(notFoundRes, {
    'non-existent route status is 404': (r) => r.status === 404,
  });

  // Pause for a short time to simulate real user pacing
  sleep(1);
}
