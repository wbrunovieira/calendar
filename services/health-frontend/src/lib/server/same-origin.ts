/**
 * True when a state-changing request provably comes from this app's own pages: the browser says
 * `Sec-Fetch-Site: same-origin`, or (older browsers) sends an Origin matching the Host.
 */
export function isSameOriginRequest(req: Request): boolean {
  const site = req.headers.get('sec-fetch-site');
  if (site) return site === 'same-origin';

  const origin = req.headers.get('origin');
  if (!origin) return false;
  const host = req.headers.get('host') ?? new URL(req.url).host;
  try {
    return new URL(origin).host === host;
  } catch {
    return false;
  }
}
