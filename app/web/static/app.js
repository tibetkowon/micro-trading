/* Micro Trading - Client JS */

document.addEventListener('DOMContentLoaded', () => {

    // === Toggle limit price field based on order type ===
    const orderType = document.getElementById('order-type');
    const limitPrice = document.getElementById('limit-price');

    if (orderType && limitPrice) {
        const toggle = () => {
            limitPrice.disabled = orderType.value !== 'LIMIT';
            if (limitPrice.disabled) limitPrice.value = '';
        };
        orderType.addEventListener('change', toggle);
        toggle();
    }

    // === Fetch current price from API ===
    async function fetchPrice(symbol, market) {
        if (!symbol) return null;
        try {
            const res = await fetch(`/api/market/price/${encodeURIComponent(symbol)}?market=${encodeURIComponent(market || 'KR')}`);
            if (!res.ok) return null;
            return await res.json();
        } catch {
            return null;
        }
    }

    function renderPrice(container, data) {
        if (!container) return;
        if (!data) {
            container.innerHTML = '<span class="price-error">조회 실패</span>';
            return;
        }
        if (data.price === 0) {
            container.innerHTML = '<span class="price-error">없는 종목이거나 시세 없음</span>';
            return;
        }
        const changeSign = data.change >= 0 ? '+' : '';
        const changeClass = data.change >= 0 ? 'positive' : 'negative';
        container.innerHTML =
            `<span class="price-value">${Number(data.price).toLocaleString()}</span>` +
            `<span class="price-change ${changeClass}">${changeSign}${Number(data.change).toLocaleString()} (${changeSign}${data.change_pct}%)</span>`;
    }

    function showLoading(container) {
        if (container) container.innerHTML = '<span class="price-loading">조회 중...</span>';
    }

    // === Orders page: memo select → auto-fill symbol + market + price ===
    const memoSelect = document.getElementById('memo-select');
    const symbolInput = document.getElementById('symbol');
    const marketSelect = document.getElementById('market');
    const priceDisplay = document.getElementById('price-display');

    if (memoSelect && symbolInput && marketSelect) {
        memoSelect.addEventListener('change', async () => {
            if (!memoSelect.value) return;
            const opt = memoSelect.options[memoSelect.selectedIndex];
            symbolInput.value = memoSelect.value;
            const marketVal = opt.getAttribute('data-market');
            if (marketVal) marketSelect.value = marketVal;

            if (priceDisplay) {
                showLoading(priceDisplay);
                const data = await fetchPrice(memoSelect.value, marketVal || marketSelect.value);
                renderPrice(priceDisplay, data);
            }
        });
    }

    // Orders page: fetch price when symbol input loses focus
    if (symbolInput && priceDisplay) {
        let debounceTimer;
        symbolInput.addEventListener('blur', async () => {
            const sym = symbolInput.value.trim();
            if (!sym) return;
            showLoading(priceDisplay);
            const data = await fetchPrice(sym, marketSelect ? marketSelect.value : 'KR');
            renderPrice(priceDisplay, data);
        });

        if (marketSelect) {
            marketSelect.addEventListener('change', async () => {
                const sym = symbolInput.value.trim();
                if (!sym) return;
                showLoading(priceDisplay);
                const data = await fetchPrice(sym, marketSelect.value);
                renderPrice(priceDisplay, data);
            });
        }
    }

    // === Stocks page: memo form price validation ===
    const memoSymbol = document.getElementById('memo-symbol');
    const memoMarket = document.getElementById('memo-market');
    const memoPriceBtn = document.getElementById('memo-price-btn');
    const memoPriceDisplay = document.getElementById('memo-price-display');
    const memoNameInput = document.getElementById('memo-name');

    if (memoPriceBtn && memoSymbol && memoMarket) {
        memoPriceBtn.addEventListener('click', async () => {
            const sym = memoSymbol.value.trim();
            if (!sym) { alert('종목코드를 먼저 입력해주세요.'); return; }
            showLoading(memoPriceDisplay);
            const data = await fetchPrice(sym, memoMarket.value);
            renderPrice(memoPriceDisplay, data);
            if (data && data.price === 0) {
                alert('유효하지 않은 종목코드이거나 현재가를 조회할 수 없습니다.');
            }
        });
    }

    // === Stocks page: recommended stock price query buttons ===
    document.querySelectorAll('.btn-fetch-price').forEach(btn => {
        btn.addEventListener('click', async () => {
            const symbol = btn.dataset.symbol;
            const market = btn.dataset.market;
            const target = document.getElementById(`price-${market}-${symbol}`);
            if (!target) return;
            btn.disabled = true;
            btn.textContent = '조회 중...';
            const data = await fetchPrice(symbol, market);
            renderPrice(target, data);
            btn.textContent = '현재가';
            btn.disabled = false;
        });
    });

    // === Strategies page: append symbol from memo dropdown ===
    const stratMemoSelect = document.getElementById('strat-memo-select');
    const stratSymbolsInput = document.getElementById('strat-symbols');

    if (stratMemoSelect && stratSymbolsInput) {
        stratMemoSelect.addEventListener('change', () => {
            if (!stratMemoSelect.value) return;
            const current = stratSymbolsInput.value.trim();
            const symbols = current ? current.split(',').map(s => s.trim()).filter(Boolean) : [];
            if (!symbols.includes(stratMemoSelect.value)) {
                symbols.push(stratMemoSelect.value);
            }
            stratSymbolsInput.value = symbols.join(',');
            stratMemoSelect.value = '';
        });
    }
});
