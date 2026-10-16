/**
 * Listens to a review's change notices, as an app does.
 *
 * notices(address, id) opens the review's events route on the server at
 * `address` and returns {next(wait), close()}: next resolves with the next
 * notice, or rejects once `wait` milliseconds pass without one.
 */
import { request } from 'node:http';

export function notices(address, id) {
  const url = new URL(`/api/reviews/${id}/events`, address);
  const queue = [];
  const waiting = [];
  let buffer = '';
  const req = request({ host: url.hostname, port: url.port, path: url.pathname, headers: { Host: url.host, Accept: 'text/event-stream' } }, res => {
    res.setEncoding('utf8');
    res.on('data', chunk => {
      buffer += chunk;
      let end;
      while ((end = buffer.indexOf('\n\n')) >= 0) {
        const event = buffer.slice(0, end);
        buffer = buffer.slice(end + 2);
        const data = event.split('\n').filter(l => l.startsWith('data: ')).map(l => l.slice(6)).join('\n');
        if (!data) continue;
        const notice = JSON.parse(data);
        if (waiting.length) waiting.shift()(notice); else queue.push(notice);
      }
    });
  });
  req.on('error', () => {});
  req.end();
  return {
    next(wait = 3000) {
      if (queue.length) return Promise.resolve(queue.shift());
      return new Promise((resolve, reject) => {
        const timer = setTimeout(() => {
          waiting.splice(waiting.indexOf(take), 1);
          reject(new Error(`no notice came in ${wait} ms`));
        }, wait);
        const take = notice => { clearTimeout(timer); resolve(notice); };
        waiting.push(take);
      });
    },
    close: () => req.destroy(),
  };
}
