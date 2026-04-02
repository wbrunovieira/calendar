import { NextRequest, NextResponse } from 'next/server';

const MAGIC_FORMULA_URL =
  process.env.MAGIC_FORMULA_URL || 'http://magic-formula:8000';

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);
  const endpoint = searchParams.get('endpoint') || 'recommendations';
  const top = searchParams.get('top');
  const ticker = searchParams.get('ticker');

  let url: string;
  switch (endpoint) {
    case 'recommendations':
      url = `${MAGIC_FORMULA_URL}/api/recommendations${top ? `?top=${top}` : ''}`;
      break;
    case 'backtest':
      url = `${MAGIC_FORMULA_URL}/api/backtest`;
      break;
    case 'ranking-history':
      if (!ticker) {
        return NextResponse.json({ error: 'ticker required' }, { status: 400 });
      }
      url = `${MAGIC_FORMULA_URL}/api/ranking-history/${encodeURIComponent(ticker)}`;
      break;
    default:
      return NextResponse.json({ error: 'invalid endpoint' }, { status: 400 });
  }

  try {
    const res = await fetch(url, { cache: 'no-store' });
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch (e) {
    console.error('Magic Formula proxy error:', e);
    return NextResponse.json(
      { error: 'Failed to reach magic-formula service' },
      { status: 502 }
    );
  }
}
