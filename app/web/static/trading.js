/* Micro Trading - Trading View JS */

// === Watchlist Tab Switching ===
function switchWlTab(btn) {
    const tab = btn.dataset.tab;

    // Update tab buttons
    document.querySelectorAll('.wl-tab').forEach(t => t.classList.remove('active'));
    btn.classList.add('active');

    // Show/hide tab content
    document.querySelectorAll('.wl-tab-content').forEach(c => c.style.display = 'none');
    const content = document.getElementById('wl-content-' + tab);
    if (content) content.style.display = '';

    // Hide search results when switching tabs
    const sr = document.getElementById('wl-search-results');
    if (sr) sr.style.display = 'none';

    // Clear search
    const input = document.getElementById('wl-search-input');
    if (input) input.value = '';
}

// === Watchlist search show/hide ===
document.addEventListener('htmx:afterSwap', function(evt) {
    if (evt.detail.target && evt.detail.target.id === 'wl-search-results') {
        const sr = document.getElementById('wl-search-results');
        const tabs = document.getElementById('wl-tabs-area');
        if (sr && sr.innerHTML.trim()) {
            sr.style.display = '';
            if (tabs) tabs.style.display = 'none';
        } else {
            sr.style.display = 'none';
            if (tabs) tabs.style.display = '';
        }
    }
});

// Handle empty search (show tabs again)
document.addEventListener('DOMContentLoaded', function() {
    const searchInput = document.getElementById('wl-search-input');
    if (searchInput) {
        searchInput.addEventListener('input', function() {
            if (!this.value.trim()) {
                const sr = document.getElementById('wl-search-results');
                const tabs = document.getElementById('wl-tabs-area');
                if (sr) { sr.style.display = 'none'; sr.innerHTML = ''; }
                if (tabs) tabs.style.display = '';
            }
        });
    }
});

function clearSearch() {
    const sr = document.getElementById('wl-search-results');
    const tabs = document.getElementById('wl-tabs-area');
    const input = document.getElementById('wl-search-input');
    if (sr) { sr.style.display = 'none'; sr.innerHTML = ''; }
    if (tabs) tabs.style.display = '';
    if (input) input.value = '';
}

// === Mark selected item in watchlist ===
function markSelected(el) {
    document.querySelectorAll('.wl-item.selected').forEach(i => i.classList.remove('selected'));
    el.classList.add('selected');
}

// === Order Form: Side Toggle ===
function setSide(side, btn) {
    // Update hidden input
    const input = document.getElementById('order-side');
    if (input) input.value = side;

    // Toggle button state
    document.querySelectorAll('.side-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');

    // Update submit button text and style
    const submitBtn = document.getElementById('order-submit-btn');
    if (submitBtn) {
        if (side === 'BUY') {
            submitBtn.textContent = '매수 주문';
            submitBtn.className = 'order-submit-btn buy';
        } else {
            submitBtn.textContent = '매도 주문';
            submitBtn.className = 'order-submit-btn sell';
        }
    }
}

// === Order Form: Limit Price Toggle ===
function toggleLimitPrice(select) {
    const limitInput = document.getElementById('inline-limit-price');
    if (!limitInput) return;
    if (select.value === 'LIMIT') {
        limitInput.disabled = false;
        limitInput.placeholder = '지정가 입력';
    } else {
        limitInput.disabled = true;
        limitInput.value = '';
        limitInput.placeholder = '시장가 주문';
    }
}

// === Order completion: clear result after delay, refresh sidebar ===
function onOrderComplete(evt) {
    // Auto-hide result message after 5 seconds
    setTimeout(() => {
        const result = document.getElementById('order-result');
        if (result) result.innerHTML = '';
    }, 5000);
}

// === Listen for refreshSidebar event (triggered by order submit) ===
document.addEventListener('refreshSidebar', function() {
    // Trigger refresh on sidebar sections
    htmx.trigger(document.body, 'refreshSidebar');
});

// Listen for the HX-Trigger header
document.body.addEventListener('refreshSidebar', function() {
    // Re-fetch sidebar partials
    document.querySelectorAll('[hx-get*="portfolio-compact"], [hx-get*="positions-compact"], [hx-get*="orders-compact"]').forEach(el => {
        htmx.trigger(el, 'load');
    });
    // Also refresh stock position if visible
    const stockPos = document.getElementById('stock-position');
    if (stockPos) {
        htmx.trigger(stockPos, 'load');
    }
});

// === Clean up chart when navigating away ===
document.addEventListener('htmx:beforeSwap', function(evt) {
    if (evt.detail.target && evt.detail.target.id === 'panel-main') {
        if (window._dailyChart) {
            window._dailyChart.destroy();
            window._dailyChart = null;
        }
        if (window._dashChart) {
            window._dashChart.destroy();
            window._dashChart = null;
        }
    }
});
