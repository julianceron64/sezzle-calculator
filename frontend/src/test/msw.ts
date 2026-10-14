import { setupServer } from 'msw/node';

// Node's fetch cannot resolve relative URLs, so tests run against an absolute
// API origin (injected via test.env in vite.config.ts).
export const API_ORIGIN = 'http://api.test';

export const apiUrl = (operation: string) =>
  `${API_ORIGIN}/api/v1/${operation}`;

export const server = setupServer();
