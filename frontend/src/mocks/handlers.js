import { http, HttpResponse } from 'msw'

// ── 서버 상태 ──
const serverStatus = {
  market_open: true,
  us_market_open: false,
  ws_connected: true,
  monitored_count: 2,
  trader_state: 'MONITORING',
  trader_state_us: 'IDLE',
  available_cash: 1234567,
}

// ── 잔고 ──
const balance = {
  total_eval: 5432100,
  withdrawable_amount: 3210000,
  asset_change_amt: 87600,
  asset_change_rate: 1.64,
}

// ── 보유 종목 ──
const positions = [
  {
    pdno: '005930',
    prdt_name: '삼성전자',
    market: 'KR',
    hldg_qty: 10,
    pchs_avg_pric: 72000,
    prpr: 75400,
    evlu_erng_rt: 4.72,
    evlu_pfls_amt: 34000,
  },
  {
    pdno: '035420',
    prdt_name: 'NAVER',
    market: 'KR',
    hldg_qty: 3,
    pchs_avg_pric: 185000,
    prpr: 181000,
    evlu_erng_rt: -2.16,
    evlu_pfls_amt: -12000,
  },
  {
    pdno: 'NVDA',
    prdt_name: 'NVIDIA',
    market: 'US',
    hldg_qty: 5,
    pchs_avg_pric: 420.5,
    prpr: 487.3,
    evlu_erng_rt: 15.88,
    evlu_pfls_amt: 334.0,
  },
]

// ── 모니터링 포지션 ──
const monitorPositions = [
  {
    stock_code: '005930',
    stock_name: '삼성전자',
    market: 'KR',
    filled_price: 72000,
    target_price: 74160,
    stop_price: 70560,
    created_at: '2026-03-19T09:25:00Z',
  },
  {
    stock_code: '035720',
    stock_name: '카카오',
    market: 'KR',
    filled_price: 48500,
    target_price: 49955,
    stop_price: 47530,
    created_at: '2026-03-19T10:12:00Z',
  },
]

// ── 주문 내역 ──
const orders = [
  {
    id: 1,
    stock_code: '005930',
    stock_name: '삼성전자',
    market: 'KR',
    order_type: 'BUY',
    qty: 10,
    price: 72000,
    filled_price: 72000,
    status: 'FILLED',
    source: 'AGENT',
    sell_reason: null,
    created_at: '2026-03-19T09:15:23Z',
  },
  {
    id: 2,
    stock_code: '035720',
    stock_name: '카카오',
    market: 'KR',
    order_type: 'BUY',
    qty: 5,
    price: 48500,
    filled_price: 48500,
    status: 'FILLED',
    source: 'AGENT',
    sell_reason: null,
    created_at: '2026-03-19T10:12:05Z',
  },
  {
    id: 3,
    stock_code: '035420',
    stock_name: 'NAVER',
    market: 'KR',
    order_type: 'BUY',
    qty: 3,
    price: 185000,
    filled_price: 185000,
    status: 'FILLED',
    source: 'AGENT',
    sell_reason: null,
    created_at: '2026-03-18T09:20:00Z',
  },
  {
    id: 4,
    stock_code: '000660',
    stock_name: 'SK하이닉스',
    market: 'KR',
    order_type: 'SELL',
    qty: 7,
    price: 183000,
    filled_price: 183000,
    status: 'FILLED',
    source: 'AGENT',
    sell_reason: '목표가 도달',
    created_at: '2026-03-18T14:30:00Z',
  },
  {
    id: 5,
    stock_code: 'NVDA',
    stock_name: 'NVIDIA',
    market: 'US',
    order_type: 'BUY',
    qty: 5,
    price: 420.5,
    filled_price: 421.0,
    status: 'FILLED',
    source: 'AGENT',
    sell_reason: null,
    created_at: '2026-03-17T23:30:00Z',
  },
  {
    id: 6,
    stock_code: '207940',
    stock_name: '삼성바이오로직스',
    market: 'KR',
    order_type: 'BUY',
    qty: 1,
    price: 820000,
    filled_price: 0,
    status: 'PENDING',
    source: 'AGENT',
    sell_reason: null,
    created_at: '2026-03-19T10:45:00Z',
  },
  {
    id: 7,
    stock_code: '041510',
    stock_name: 'SM엔터테인먼트',
    market: 'KR',
    order_type: 'BUY',
    qty: 10,
    price: 95000,
    filled_price: 0,
    status: 'CANCELLED',
    source: 'AGENT',
    sell_reason: null,
    created_at: '2026-03-18T11:00:00Z',
  },
]

// ── 서비스 로그 ──
const serviceLogs = [
  {
    id: 1,
    level: 'ERROR',
    source: 'TRADER',
    message: '국장 트레이더 시작 — MONITORING 상태 진입 (종목: 삼성전자)',
    detail: null,
    timestamp: '2026-03-19T09:25:10Z',
  },
  {
    id: 2,
    level: 'ERROR',
    source: 'TRADER',
    message: '종목 선정 완료 — Claude 추천: 005930 삼성전자 (score: 8.5/10)',
    detail: JSON.stringify({ stock_code: '005930', reason: '거래량 급등 + RSI 50 이하 저평가 구간' }, null, 2),
    timestamp: '2026-03-19T09:20:05Z',
  },
  {
    id: 3,
    level: 'ERROR',
    source: 'TRADER',
    message: 'KIS API 오류: 잔고 조회 실패 (rt_cd=1)',
    detail: JSON.stringify({ error_code: 'EGW00123', message: '시스템 오류가 발생하였습니다', endpoint: '/uapi/domestic-stock/v1/trading/inquire-balance' }, null, 2),
    timestamp: '2026-03-19T09:18:30Z',
  },
  {
    id: 4,
    level: 'ERROR',
    source: 'MONITOR',
    message: '포지션 가격 미갱신 경고 — 카카오 (35720) 5분 이상 WebSocket 데이터 없음',
    detail: null,
    timestamp: '2026-03-19T10:30:00Z',
  },
  {
    id: 5,
    level: 'ERROR',
    source: 'SYSTEM',
    message: 'KIS Access Token 갱신 완료 (만료까지 20시간)',
    detail: null,
    timestamp: '2026-03-19T09:00:00Z',
  },
]

// ── KIS API 에러 로그 ──
const kisLogs = [
  {
    id: 1,
    error_code: 'EGW00123',
    endpoint: '/uapi/domestic-stock/v1/trading/inquire-balance',
    error_message: '시스템 오류가 발생하였습니다',
    raw_response: '{"rt_cd":"1","msg_cd":"EGW00123","msg1":"시스템 오류가 발생하였습니다","output":null}',
    timestamp: '2026-03-19T09:18:30Z',
  },
  {
    id: 2,
    error_code: 'APBK0013',
    endpoint: '/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice',
    error_message: '조회 가능한 데이터가 없습니다',
    raw_response: '{"rt_cd":"1","msg_cd":"APBK0013","msg1":"조회 가능한 데이터가 없습니다"}',
    timestamp: '2026-03-18T14:22:00Z',
  },
]

// ── 순위 로그 ──
const rankingLogs = [
  {
    id: 2,
    market: 'KR',
    ranking_types: JSON.stringify(['volume', 'strength']),
    ranking_condition: 'AND',
    volume_count: 15,
    strength_count: 12,
    exec_count_count: -1,
    disparity_count: -1,
    result_stocks: JSON.stringify([
      { stock_code: '005930', stock_name: '삼성전자' },
      { stock_code: '035720', stock_name: '카카오' },
      { stock_code: '000660', stock_name: 'SK하이닉스' },
    ]),
    filtered_stocks: JSON.stringify([
      { stock_code: '005380', stock_name: '현대차', filter_reason: 'RSI 과매수', rsi14: 78.3 },
      { stock_code: '051910', stock_name: 'LG화학', filter_reason: '5분 이격도 초과', disparity_m5: 4.2 },
    ]),
    error_message: null,
    created_at: '2026-03-19T09:15:00Z',
  },
  {
    id: 1,
    market: 'KR',
    ranking_types: JSON.stringify(['volume', 'strength', 'disparity']),
    ranking_condition: 'OR',
    volume_count: 20,
    strength_count: 18,
    exec_count_count: -1,
    disparity_count: 11,
    result_stocks: JSON.stringify([
      { stock_code: '000660', stock_name: 'SK하이닉스' },
      { stock_code: '041510', stock_name: 'SM엔터테인먼트' },
    ]),
    filtered_stocks: JSON.stringify([]),
    error_message: null,
    created_at: '2026-03-18T09:15:00Z',
  },
]

// ── 종목 선정 로그 ──
const selectionLogs = [
  {
    id: 2,
    market: 'KR',
    ranking_log_id: 2,
    selected_stock_code: '005930',
    selected_stock_name: '삼성전자',
    candidate_stocks: JSON.stringify([
      { stock_code: '005930', stock_name: '삼성전자', score: 8.5, reason: '거래량 급등, RSI 저평가' },
      { stock_code: '035720', stock_name: '카카오', score: 7.2, reason: '체결강도 강세' },
      { stock_code: '000660', stock_name: 'SK하이닉스', score: 6.8, reason: '이격도 양호' },
    ]),
    llm_reasoning: '삼성전자는 금일 거래량이 전일 대비 230% 증가했으며, RSI(14)=48로 과매수 구간이 아닙니다. 반도체 업황 회복 기대감과 외국인 순매수가 집중되고 있어 단기 상승 여력이 있다고 판단합니다.',
    error_message: null,
    created_at: '2026-03-19T09:20:00Z',
  },
  {
    id: 1,
    market: 'KR',
    ranking_log_id: 1,
    selected_stock_code: '000660',
    selected_stock_name: 'SK하이닉스',
    candidate_stocks: JSON.stringify([
      { stock_code: '000660', stock_name: 'SK하이닉스', score: 9.1, reason: '강한 상승 모멘텀' },
      { stock_code: '041510', stock_name: 'SM엔터테인먼트', score: 5.9, reason: '거래량 증가' },
    ]),
    llm_reasoning: 'SK하이닉스는 HBM 수요 증가와 함께 외국인 매수세가 집중되고 있습니다. 체결강도 145로 매수 우세를 보이며 20일 이격도도 양호한 구간입니다.',
    error_message: null,
    created_at: '2026-03-18T09:20:00Z',
  },
]

// ── 설정 ──
const settings = {
  trading_enabled: true,
  trading_start_time: '09:15',
  trading_end_time: '15:15',
  ranking_excl_cls: '1111111111',
  ranking_types: ['volume', 'strength'],
  ranking_price_min: '5000',
  ranking_price_max: '100000',
  ranking_top_n: 20,
  ranking_condition: 'AND',
  ranking_volume_min_incrrate: 0,
  ranking_strength_min: 100,
  ranking_execcount_net_buy_only: true,
  ranking_disparity_d20_min: 0,
  ranking_disparity_d20_max: 0,
  max_positions: 2,
  order_amount_pct: 95,
  take_profit_pct: 3.0,
  stop_loss_pct: 2.0,
  sell_conditions: ['target_pct', 'stop_pct', 'rsi_overbought'],
  indicator_check_interval_min: 5,
  indicator_rsi_sell_threshold: 70,
  indicator_macd_bearish_sell: false,
  stagnation_threshold_pct: 1.0,
  stagnation_duration_min: 30,
  min_trading_value: 5000000000,
  buy_pause_start: '11:00',
  buy_pause_end: '14:00',
  trailing_trigger_pct: 2.0,
  trailing_stop_pct: 1.0,
  daily_max_loss_pct: 3.0,
  index_codes: ['0001'],
  claude_model: 'claude-sonnet-4-6',
  filter_rsi_max: 80,
  filter_disparity_m5_max: 3.0,
  filter_high_price_diff_min: -5.0,
  filter_open_price_diff_max: 20.0,
  index_drop_threshold_pct: -1.0,
  us_trading_enabled: true,
  us_dst_enabled: true,
  us_trading_start_time: '22:30',
  us_trading_end_time: '05:00',
  us_ranking_types: ['volume'],
  us_ranking_exchange: 'NAS',
  us_ranking_price_min: '10',
  us_ranking_price_max: '500',
  us_ranking_vol_rang: 0,
  us_ranking_top_n: 20,
  us_daily_max_loss_pct: 0,
  us_min_trading_value: 0,
  ws_connected: true,
}

export const handlers = [
  http.get('/api/server/status', () => HttpResponse.json(serverStatus)),
  http.get('/api/balance', () => HttpResponse.json(balance)),
  http.get('/api/positions', () => HttpResponse.json({ positions })),

  http.post('/api/ws/connect', () => HttpResponse.json({ message: 'WebSocket 연결됨' })),
  http.post('/api/ws/disconnect', () => HttpResponse.json({ message: 'WebSocket 해제됨' })),

  http.get('/api/monitor/positions', () => HttpResponse.json({ positions: monitorPositions })),
  http.delete('/api/monitor/positions/:code', () => HttpResponse.json({ message: '해제됨' })),

  http.get('/api/orders', () => HttpResponse.json({ orders })),
  http.post('/api/orders/:id/cancel', () => HttpResponse.json({ message: '취소됨' })),
  http.delete('/api/orders/:id', () => HttpResponse.json({ message: '삭제됨' })),

  http.get('/api/logs/service', ({ request }) => {
    const url = new URL(request.url)
    const source = url.searchParams.get('source') || 'ALL'
    const filtered = source === 'ALL' ? serviceLogs : serviceLogs.filter(l => l.source === source)
    return HttpResponse.json({ logs: filtered })
  }),

  http.get('/api/logs/kis', () => HttpResponse.json({ logs: kisLogs })),
  http.delete('/api/logs/kis/:id', () => HttpResponse.json({ message: '삭제됨' })),

  http.get('/api/logs/ranking', () => HttpResponse.json({ logs: rankingLogs })),
  http.get('/api/logs/selection', () => HttpResponse.json({ logs: selectionLogs })),

  http.get('/api/settings', () => HttpResponse.json(settings)),
  http.patch('/api/settings', () => HttpResponse.json({ message: '설정이 저장되었습니다.' })),
]
