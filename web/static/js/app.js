/**
 * GOLD SOCKET - Modern Dashboard v2.0
 * Real-time USD Exchange Rate & Gold Price Monitor
 * Using WebSocket for real-time updates
 */

(function() {
    'use strict';

    // ===== STATE =====
    const state = {
        ws: null,
        reconnectTimer: null,
        reconnectAttempts: 0,
        maxReconnectAttempts: 10,
        reconnectDelay: 1000, // Start with 1 second
        isConnected: false,
        jsonVisible: false,
        manualUpdateOpen: false,
        theme: 'light',
        hasEverConnected: false,
        lastConnectToast: 0,
        lastDisconnectAt: null,
        intentionalClose: false,
        lastUSDData: null,
        lastMarketData: null,
        rateHistory: [], // For sparkline
        bannerDismissed: false, // Track if user dismissed the banner
        marketOpen: true,
        stats: {
            updates: 0,
            high: null,
            low: null,
            sum: 0,
            count: 0
        }
    };

    // ===== CONFIG =====
    const config = {
        // Determine WebSocket URL based on current protocol
        get wsUrl() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const host = window.location.host;
            return `${protocol}//${host}/ws`;
        },
        maxHistoryLength: 50, // Max points for sparkline
        reconnectBackoffMultiplier: 1.5, // Exponential backoff
        maxReconnectDelay: 30000, // Max 30 seconds
        connectToastCooldownMs: 30000,
        reconnectToastMinDisconnectMs: 5000
    };

    // ===== DOM ELEMENTS =====
    const elements = {};

    // ===== INITIALIZATION =====
    function init() {
        cacheElements();
        loadTheme();
        setupEventListeners();
        connectWebSocket(); // Auto-connect WebSocket
    }

    function cacheElements() {
        // Status elements
        elements.navbarStatus = document.getElementById('navbar-status');
        elements.connectionStatus = document.getElementById('connection-status');
        elements.statusTextLarge = document.getElementById('status-text-large');

        // USD Card elements
        elements.usdBuy = document.getElementById('usd-buy');
        elements.usdSell = document.getElementById('usd-sell');
        elements.usdSpread = document.getElementById('usd-spread');
        elements.rateChange = document.getElementById('rate-change');
        elements.usdTime = document.getElementById('usd-time');
        elements.usdMarketStatus = document.getElementById('usd-market-status');
        elements.sparkline = document.getElementById('usd-sparkline');

        // Stats elements
        elements.statUpdates = document.getElementById('stat-updates');
        elements.statHigh = document.getElementById('stat-high');
        elements.statLow = document.getElementById('stat-low');
        elements.statAvg = document.getElementById('stat-avg');

        // Market elements
        elements.marketGrid = document.getElementById('market-grid');
        elements.marketStatusBadge = document.getElementById('market-status-badge');

        // Form elements
        elements.updateFormWrapper = document.getElementById('update-form-wrapper');
        elements.manualToggleText = document.getElementById('manual-toggle-text');
        elements.manualToggleIcon = document.getElementById('manual-toggle-icon');
        elements.manualBuy = document.getElementById('manual-buy');
        elements.manualSell = document.getElementById('manual-sell');

        // JSON elements
        elements.jsonSection = document.getElementById('json-section');
        elements.jsonOutput = document.getElementById('json-output');

        // Footer
        elements.lastUpdate = document.getElementById('last-update');

        // Toast container
        elements.toastContainer = document.getElementById('toast-container');

        // Market closed banner
        elements.marketClosedBanner = document.getElementById('market-closed-banner');
        elements.bannerSubtitle = document.getElementById('banner-subtitle');

        // USD Card (for closed styling)
        elements.usdCard = document.querySelector('.usd-card');
    }

    function setupEventListeners() {
        document.addEventListener('visibilitychange', handleVisibilityChange);
        window.addEventListener('beforeunload', disconnectWebSocket);
        window.addEventListener('online', handleOnline);
    }

    function handleVisibilityChange() {
        // When tab becomes visible, check connection
        if (!document.hidden) {
            if (!state.isConnected) {
                connectWebSocket();
            }
        }
    }

    function handleOnline() {
        // Reconnect when browser comes back online
        if (!state.isConnected) {
            connectWebSocket();
        }
    }

    // ===== WEBSOCKET CONNECTION =====
    function connectWebSocket() {
        // Clear any existing reconnect timer
        if (state.reconnectTimer) {
            clearTimeout(state.reconnectTimer);
            state.reconnectTimer = null;
        }

        // Close existing connection if any
        if (state.ws && state.ws.readyState === WebSocket.OPEN) {
            state.ws.close();
        }

        try {
            state.intentionalClose = false;
            state.ws = new WebSocket(config.wsUrl);

            // Connection opened
            state.ws.onopen = handleWebSocketOpen;

            // Message received
            state.ws.onmessage = handleWebSocketMessage;

            // Connection closed
            state.ws.onclose = handleWebSocketClose;

            // Connection error
            state.ws.onerror = handleWebSocketError;

            // Connection timeout
            setTimeout(() => {
                if (state.ws && state.ws.readyState === WebSocket.CONNECTING) {
                    console.warn('WebSocket connection timeout');
                    state.ws.close();
                }
            }, 10000);

        } catch (error) {
            console.error('WebSocket connection error:', error);
            scheduleReconnect();
        }
    }

    function handleWebSocketOpen(event) {
        console.log('WebSocket connected');
        state.isConnected = true;
        state.reconnectAttempts = 0;
        state.reconnectDelay = 1000;
        const now = Date.now();

        updateConnectionStatus(true);

        // Show connect toast only on first connect or after a meaningful disconnect
        if (!state.hasEverConnected) {
            showToast('success', 'เชื่อมต่อสำเร็จ', 'รับข้อมูลแบบ Real-time');
            state.hasEverConnected = true;
            state.lastConnectToast = now;
        } else if (state.lastDisconnectAt) {
            const downFor = now - state.lastDisconnectAt;
            const cooldownOk = now - state.lastConnectToast >= config.connectToastCooldownMs;
            if (downFor >= config.reconnectToastMinDisconnectMs && cooldownOk) {
                showToast('success', 'เชื่อมต่ออีกครั้ง', 'กลับมารับข้อมูลแบบ Real-time');
                state.lastConnectToast = now;
            }
        }
        state.lastDisconnectAt = null;

        // Auto-fetch data when connected
        setTimeout(() => {
            fetchData();
            fetchMarketData();
        }, 100);
    }

    function handleWebSocketMessage(event) {
        try {
            const message = event.data;

            // Try to parse as JSON
            const data = JSON.parse(message);

            // Check message type - all wrapped format
            if (data.type === 'usd_rate') {
                handleUSDRate(data.data);
            } else if (data.type === 'market_data') {
                handleMarketData(data.data);
            } else if (data.type) {
                // Other wrapped message types
                console.log('Received message type:', data.type);
            } else {
                // Fallback: direct data (backward compatibility)
                handleUSDRate(data);
            }

        } catch (error) {
            // Not JSON, log it
            console.log('WebSocket message:', event.data);
        }
    }

    function handleWebSocketClose(event) {
        console.log('WebSocket disconnected:', event.code, event.reason);
        state.isConnected = false;
        updateConnectionStatus(false);

        if (event.code !== 1000 && !state.intentionalClose) {
            state.lastDisconnectAt = Date.now();
        }

        // Attempt to reconnect if not closed intentionally
        if (event.code !== 1000) {
            scheduleReconnect();
        }
    }

    function handleWebSocketError(event) {
        console.error('WebSocket error:', event);
    }

    function scheduleReconnect() {
        if (state.reconnectAttempts >= state.maxReconnectAttempts) {
            console.log('Max reconnection attempts reached');
            showToast('error', 'Disconnected', 'Connection lost. Please refresh the page.');
            return;
        }

        state.reconnectAttempts++;

        // Calculate delay with exponential backoff
        const delay = Math.min(
            state.reconnectDelay * Math.pow(config.reconnectBackoffMultiplier, state.reconnectAttempts - 1),
            config.maxReconnectDelay
        );

        console.log(`Reconnecting in ${delay}ms (attempt ${state.reconnectAttempts}/${state.maxReconnectAttempts})`);

        state.reconnectTimer = setTimeout(() => {
            connectWebSocket();
        }, delay);
    }

    function disconnectWebSocket() {
        // Clear reconnect timer
        if (state.reconnectTimer) {
            clearTimeout(state.reconnectTimer);
            state.reconnectTimer = null;
        }

        // Close WebSocket
        if (state.ws) {
            state.intentionalClose = true;
            state.ws.close(1000, 'User disconnected');
            state.ws = null;
        }

        state.isConnected = false;
        updateConnectionStatus(false);
    }

    function updateConnectionStatus(connected) {
        // Update navbar status badge
        if (elements.navbarStatus) {
            elements.navbarStatus.className = `status-badge ${connected ? 'connected' : ''}`;
            elements.navbarStatus.querySelector('.status-label').textContent =
                connected ? 'Connected' : 'Disconnected';
        }

        // Update connection status card
        if (elements.connectionStatus) {
            elements.connectionStatus.className = `status-indicator-large ${connected ? 'connected' : 'disconnected'}`;
        }
        if (elements.statusTextLarge) {
            elements.statusTextLarge.textContent = connected ? 'Connected' : 'Disconnected';
        }
    }

    // ===== DATA HANDLERS =====
    function handleUSDRate(data) {
        if (!data) return;

        // Check if data actually changed
        if (state.lastUSDData && JSON.stringify(state.lastUSDData) === JSON.stringify(data)) {
            return;
        }

        state.lastUSDData = data;
        updateUSDRate(data);
        updateJsonOutput(data);
        updateLastUpdate(data.timestamp);
    }

    function handleMarketData(data) {
        if (!data) return;

        // Check if data actually changed
        if (state.lastMarketData && JSON.stringify(state.lastMarketData) === JSON.stringify(data)) {
            return;
        }

        state.lastMarketData = data;
        updateMarketData(data, true);
        updateJsonOutput(data);
        updateLastUpdate(data.timestamp);
    }

    // ===== USD RATE UPDATE =====
    function updateUSDRate(rate) {
        if (!rate) return;

        const previousBuy = state.lastUSDData?.buy || rate.buy;

        // Update buy rate
        if (elements.usdBuy) {
            const newDisplay = rate.buy.toFixed(2);
            if (elements.usdBuy.textContent !== newDisplay) {
                animateNumber(elements.usdBuy, parseFloat(elements.usdBuy.textContent) || 0, rate.buy);
            }
        }

        // Update sell rate
        if (elements.usdSell) {
            const newDisplay = rate.sell.toFixed(2);
            if (elements.usdSell.textContent !== newDisplay) {
                animateNumber(elements.usdSell, parseFloat(elements.usdSell.textContent) || 0, rate.sell);
            }
        }

        // Update spread
        const spread = rate.sell - rate.buy;
        if (elements.usdSpread) {
            elements.usdSpread.textContent = spread.toFixed(2);
        }

        // Update rate change indicator
        if (elements.rateChange && rate.buy !== previousBuy) {
            const diff = rate.buy - previousBuy;
            const percent = ((diff / previousBuy) * 100).toFixed(2);
            const isPositive = diff >= 0;

            elements.rateChange.className = `rate-change ${isPositive ? 'positive' : 'negative'}`;
            elements.rateChange.innerHTML = `
                <span class="change-icon">${isPositive ? '▲' : '▼'}</span>
                <span class="change-value">${isPositive ? '+' : ''}${percent}%</span>
            `;
        }

        // Update time
        if (elements.usdTime) {
            elements.usdTime.textContent = rate.time;
        }

        // Update market status badge
        if (elements.usdMarketStatus) {
            const isOpen = rate.market_status === 'open';
            elements.usdMarketStatus.className = `market-badge ${isOpen ? 'open' : ''}`;
            elements.usdMarketStatus.querySelector('span:not(.badge-dot)').textContent =
                isOpen ? 'Open' : 'Closed';
        }

        // Update market status UI (banner, card styling)
        updateMarketStatusUI(rate.market_status, rate.timestamp);

        // Update sparkline data
        updateSparkline(rate.buy);

        // Update stats
        updateStats(rate.buy);
    }

    // ===== SPARKLINE =====
    function updateSparkline(buyRate) {
        state.rateHistory.push(buyRate);
        if (state.rateHistory.length > config.maxHistoryLength) {
            state.rateHistory.shift();
        }
        renderSparkline();
    }

    function renderSparkline() {
        if (!elements.sparkline || state.rateHistory.length < 2) return;

        const data = state.rateHistory;
        const min = Math.min(...data);
        const max = Math.max(...data);
        const range = max - min || 1;

        const width = 200;
        const height = 40;
        const padding = 2;

        const points = data.map((value, index) => {
            const x = (index / (data.length - 1)) * width;
            const y = height - ((value - min) / range) * (height - padding * 2) - padding;
            return `${x},${y}`;
        });

        const linePath = `M ${points.join(' L ')}`;
        const areaPath = `${linePath} L ${width},${height} L 0,${height} Z`;

        const line = elements.sparkline.querySelector('.sparkline-line');
        const area = elements.sparkline.querySelector('.sparkline-area');

        if (line) line.setAttribute('d', linePath);
        if (area) area.setAttribute('d', areaPath);
    }

    // ===== STATS =====
    function updateStats(buyRate) {
        state.stats.updates++;
        state.stats.count++;
        state.stats.sum += buyRate;

        if (state.stats.high === null || buyRate > state.stats.high) {
            state.stats.high = buyRate;
        }
        if (state.stats.low === null || buyRate < state.stats.low) {
            state.stats.low = buyRate;
        }

        const avg = state.stats.sum / state.stats.count;

        if (elements.statUpdates) {
            elements.statUpdates.textContent = state.stats.updates;
            elements.statUpdates.classList.add('count-up');
            setTimeout(() => elements.statUpdates.classList.remove('count-up'), 300);
        }
        if (elements.statHigh) {
            elements.statHigh.textContent = state.stats.high?.toFixed(2) || '--';
        }
        if (elements.statLow) {
            elements.statLow.textContent = state.stats.low?.toFixed(2) || '--';
        }
        if (elements.statAvg) {
            elements.statAvg.textContent = avg.toFixed(2);
        }
    }

    // ===== NUMBER ANIMATION =====
    function animateNumber(element, from, to) {
        const duration = 300;
        const startTime = performance.now();

        function update(currentTime) {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            const eased = 1 - Math.pow(1 - progress, 3);

            const current = from + (to - from) * eased;
            element.textContent = current.toFixed(2);

            if (progress < 1) {
                requestAnimationFrame(update);
            } else {
                element.textContent = to.toFixed(2);
                element.classList.add('count-up');
                setTimeout(() => element.classList.remove('count-up'), 300);
            }
        }

        requestAnimationFrame(update);
    }

    // ===== MARKET DATA =====
    function updateMarketData(data, animate = false) {
        if (!data) return;

        const isInitialLoad = !elements.marketGrid.querySelector('.market-card');

        if (isInitialLoad) {
            renderMarketCards(data);
        } else {
            updateMarketValues(data, animate);
        }

        // Update market status badge
        if (elements.marketStatusBadge) {
            const isOpen = data.market_status === 'open';
            elements.marketStatusBadge.className = `market-status-badge ${isOpen ? 'open' : ''}`;
            elements.marketStatusBadge.querySelector('span:not(.status-dot)').textContent =
                `Market ${data.market_status.charAt(0).toUpperCase() + data.market_status.slice(1)}`;
        }
    }

    function renderMarketCards(data) {
        const cards = [
            {
                id: 'spot-usd',
                icon: '💵',
                name: 'Spot USD',
                values: [
                    { label: 'Bid', value: data.spot_usd.bid.toFixed(2), id: 'spot-bid' },
                    { label: 'Offer', value: data.spot_usd.offer.toFixed(2), id: 'spot-offer' },
                    { label: 'Spread', value: (data.spot_usd.offer - data.spot_usd.bid).toFixed(2), id: 'spot-spread' }
                ]
            },
            {
                id: 'g965b',
                icon: '🥇',
                name: 'G965B Retail',
                values: [
                    { label: 'Bid', value: formatNumber(data.g965b_retail.bid), id: 'g965b-bid' },
                    { label: 'Offer', value: formatNumber(data.g965b_retail.offer), id: 'g965b-offer' },
                    { label: 'Spread', value: formatNumber(data.g965b_retail.offer - data.g965b_retail.bid), id: 'g965b-spread' }
                ]
            },
            {
                id: 'g9999kg',
                icon: '🏆',
                name: 'G9999KG Retail',
                values: [
                    { label: 'Bid', value: formatNumber(data.g9999kg_retail.bid), id: 'g9999kg-bid' },
                    { label: 'Offer', value: formatNumber(data.g9999kg_retail.offer), id: 'g9999kg-offer' },
                    { label: 'Spread', value: formatNumber(data.g9999kg_retail.offer - data.g9999kg_retail.bid), id: 'g9999kg-spread' }
                ]
            },
            {
                id: 'g9999g',
                icon: '✨',
                name: 'G9999G',
                values: [
                    { label: 'Bid', value: formatNumber(data.g9999g.bid), id: 'g9999g-bid' },
                    { label: 'Offer', value: formatNumber(data.g9999g.offer), id: 'g9999g-offer' },
                    { label: 'Spread', value: formatNumber(data.g9999g.offer - data.g9999g.bid), id: 'g9999g-spread' }
                ]
            }
        ];

        elements.marketGrid.innerHTML = cards.map(card => `
            <div class="market-card" id="${card.id}-card">
                <div class="market-card-header">
                    <div class="market-card-icon">${card.icon}</div>
                    <div class="market-card-name">${card.name}</div>
                </div>
                <div class="market-data">
                    ${card.values.map(v => `
                        <div class="market-data-row">
                            <span class="label">${v.label}</span>
                            <span class="value" id="${v.id}">${v.value}</span>
                        </div>
                    `).join('')}
                </div>
            </div>
        `).join('');
    }

    function updateMarketValues(data, animate) {
        const updates = [
            { id: 'spot-bid', value: data.spot_usd.bid.toFixed(2) },
            { id: 'spot-offer', value: data.spot_usd.offer.toFixed(2) },
            { id: 'spot-spread', value: (data.spot_usd.offer - data.spot_usd.bid).toFixed(2) },
            { id: 'g965b-bid', value: formatNumber(data.g965b_retail.bid) },
            { id: 'g965b-offer', value: formatNumber(data.g965b_retail.offer) },
            { id: 'g965b-spread', value: formatNumber(data.g965b_retail.offer - data.g965b_retail.bid) },
            { id: 'g9999kg-bid', value: formatNumber(data.g9999kg_retail.bid) },
            { id: 'g9999kg-offer', value: formatNumber(data.g9999kg_retail.offer) },
            { id: 'g9999kg-spread', value: formatNumber(data.g9999kg_retail.offer - data.g9999kg_retail.bid) },
            { id: 'g9999g-bid', value: formatNumber(data.g9999g.bid) },
            { id: 'g9999g-offer', value: formatNumber(data.g9999g.offer) },
            { id: 'g9999g-spread', value: formatNumber(data.g9999g.offer - data.g9999g.bid) }
        ];

        updates.forEach(update => {
            updateValueIfChanged(update.id, update.value, animate);
        });
    }

    function updateValueIfChanged(elementId, newValue, animate) {
        const element = document.getElementById(elementId);
        if (element && element.textContent !== newValue) {
            element.textContent = newValue;
            if (animate) {
                element.classList.add('data-update');
                setTimeout(() => element.classList.remove('data-update'), 500);
            }
        }
    }

    // ===== THEME MANAGEMENT =====
    function loadTheme() {
        const savedTheme = localStorage.getItem('goldsocket-theme') || 'light';
        setTheme(savedTheme);
    }

    function setTheme(theme) {
        state.theme = theme;
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('goldsocket-theme', theme);
    }

    function toggleTheme() {
        const newTheme = state.theme === 'light' ? 'dark' : 'light';
        setTheme(newTheme);
        showToast('info', 'Theme', `Switched to ${newTheme} mode`);
    }

    // ===== TOAST NOTIFICATIONS =====
    function showToast(type, title, message) {
        if (!elements.toastContainer) return;

        const toast = document.createElement('div');
        toast.className = `toast ${type}`;

        const icons = {
            success: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>',
            error: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>',
            info: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>'
        };

        toast.innerHTML = `
            <div class="toast-icon">${icons[type] || icons.info}</div>
            <div class="toast-content">
                <div class="toast-title">${title}</div>
                <div class="toast-message">${message}</div>
            </div>
        `;

        elements.toastContainer.appendChild(toast);

        setTimeout(() => {
            toast.classList.add('removing');
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    }

    // ===== JSON OUTPUT =====
    function updateJsonOutput(data) {
        if (elements.jsonOutput) {
            elements.jsonOutput.textContent = JSON.stringify(data, null, 2);
        }
    }

    function copyJson() {
        if (elements.jsonOutput && elements.jsonOutput.textContent) {
            navigator.clipboard.writeText(elements.jsonOutput.textContent)
                .then(() => showToast('success', 'Copied', 'JSON data copied to clipboard'))
                .catch(() => showToast('error', 'Error', 'Failed to copy JSON'));
        }
    }

    function toggleJson() {
        if (!elements.jsonSection) return;
        state.jsonVisible = !state.jsonVisible;
        elements.jsonSection.classList.toggle('hidden', !state.jsonVisible);
    }

    // ===== LAST UPDATE =====
    function updateLastUpdate(timestamp) {
        if (elements.lastUpdate) {
            const date = new Date(timestamp);
            elements.lastUpdate.textContent = date.toLocaleString();
        }
    }

    // ===== MANUAL UPDATE FORM =====
    function toggleManualUpdate() {
        state.manualUpdateOpen = !state.manualUpdateOpen;

        if (elements.updateFormWrapper) {
            elements.updateFormWrapper.classList.toggle('open', state.manualUpdateOpen);
        }

        if (elements.manualToggleText) {
            elements.manualToggleText.textContent = state.manualUpdateOpen ? 'Hide' : 'Show';
        }

        if (elements.manualToggleIcon) {
            elements.manualToggleIcon.style.transform = state.manualUpdateOpen ? 'rotate(180deg)' : '';
        }
    }

    function clearManualForm() {
        if (elements.manualBuy) elements.manualBuy.value = '';
        if (elements.manualSell) elements.manualSell.value = '';
    }

    async function updateManualRate() {
        const buy = parseFloat(elements.manualBuy?.value);
        const sell = parseFloat(elements.manualSell?.value);

        if (!buy || !sell || buy <= 0 || sell <= 0) {
            showToast('error', 'Validation Error', 'Please enter valid buy and sell rates');
            return;
        }

        if (buy >= sell) {
            showToast('error', 'Validation Error', 'Buy rate should be lower than sell rate');
            return;
        }

        try {
            const response = await fetch(`/api/update-rate`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ buy, sell })
            });

            if (response.ok) {
                showToast('success', 'Success', 'USD rate updated successfully');
                toggleManualUpdate();
                clearManualForm();
                // WebSocket will receive the update automatically
            } else {
                const error = await response.text();
                showToast('error', 'Update Failed', error);
            }
        } catch (error) {
            console.error('Error updating rate:', error);
            showToast('error', 'Update Failed', error.message);
        }
    }

    // ===== MANUAL REFRESH (for debugging/testing) =====
    async function fetchData() {
        // Manual refresh - still works, but data will also come via WebSocket
        try {
            const response = await fetch(`/api/data`);
            const data = await response.json();
            handleUSDRate(data);
        } catch (error) {
            console.error('Error fetching USD rate:', error);
            showToast('error', 'Error', 'Failed to fetch USD rate');
        }
    }

    async function fetchMarketData() {
        try {
            const response = await fetch(`/api/market-data`);
            const data = await response.json();
            handleMarketData(data);
        } catch (error) {
            console.error('Error fetching market data:', error);
        }
    }

    // ===== MARKET STATUS =====
    function updateMarketStatusUI(marketStatus, lastUpdateTime) {
        const isOpen = marketStatus === 'open';
        state.marketOpen = isOpen;

        // Update USD Card styling
        if (elements.usdCard) {
            elements.usdCard.classList.toggle('market-closed', !isOpen);
        }

        // Update banner
        if (!isOpen && !state.bannerDismissed) {
            showMarketClosedBanner(lastUpdateTime);
        } else if (isOpen) {
            hideMarketClosedBanner();
            state.bannerDismissed = false; // Reset when market opens
        }
    }

    function showMarketClosedBanner(lastUpdateTime) {
        if (!elements.marketClosedBanner) return;

        // Calculate next market open
        const nextOpen = getNextMarketOpen();
        const lastUpdateStr = lastUpdateTime ? formatThaiDateTime(lastUpdateTime) : 'ไม่ทราบ';

        if (elements.bannerSubtitle) {
            elements.bannerSubtitle.textContent = `ราคาล่าสุด: ${lastUpdateStr} • เปิดทำการ: ${nextOpen}`;
        }

        elements.marketClosedBanner.classList.add('visible');
        document.body.classList.add('banner-visible');
    }

    function hideMarketClosedBanner() {
        if (!elements.marketClosedBanner) return;
        elements.marketClosedBanner.classList.remove('visible');
        document.body.classList.remove('banner-visible');
    }

    function closeBanner() {
        state.bannerDismissed = true;
        hideMarketClosedBanner();
    }

    function getNextMarketOpen() {
        const now = new Date();
        const dayOfWeek = now.getDay(); // 0 = Sunday, 6 = Saturday
        const hour = now.getHours();
        const minute = now.getMinutes();
        const currentTime = hour * 60 + minute;

        const marketOpen = 9 * 60; // 09:00
        const marketClose = 17 * 60; // 17:00

        // Weekend
        if (dayOfWeek === 0) { // Sunday
            return 'จันทร์ 09:00 น.';
        } else if (dayOfWeek === 6) { // Saturday
            return 'จันทร์ 09:00 น.';
        }

        // Weekday
        if (currentTime < marketOpen) {
            // Before market opens today
            const days = ['อาทิตย์', 'จันทร์', 'อังคาร', 'พุธ', 'พฤหัสบดี', 'ศุกร์', 'เสาร์'];
            return `${days[dayOfWeek]} 09:00 น.`;
        } else if (currentTime >= marketClose) {
            // After market closes
            if (dayOfWeek === 5) { // Friday
                return 'จันทร์ 09:00 น.';
            } else {
                const days = ['อาทิตย์', 'จันทร์', 'อังคาร', 'พุธ', 'พฤหัสบดี', 'ศุกร์', 'เสาร์'];
                return `${days[(dayOfWeek + 1) % 7]} 09:00 น.`;
            }
        }

        // During lunch break (12:00-13:00) - if applicable
        const lunchStart = 12 * 60;
        const lunchEnd = 13 * 60;
        if (currentTime >= lunchStart && currentTime < lunchEnd) {
            const days = ['อาทิตย์', 'จันทร์', 'อังคาร', 'พุธ', 'พฤหัสบดี', 'ศุกร์', 'เสาร์'];
            return `${days[dayOfWeek]} 13:00 น.`;
        }

        return 'ตอนนี้';
    }

    function formatThaiDateTime(timestamp) {
        try {
            const date = new Date(timestamp);
            return date.toLocaleString('th-TH', {
                day: 'numeric',
                month: 'short',
                hour: '2-digit',
                minute: '2-digit'
            });
        } catch {
            return timestamp;
        }
    }

    // ===== UTILITY =====
    function formatNumber(num) {
        return num.toLocaleString('en-US', {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2
        });
    }

    // ===== PUBLIC API =====
    window.GoldSocketApp = {
        connect: connectWebSocket,
        disconnect: disconnectWebSocket,
        reconnect: () => { state.reconnectAttempts = 0; connectWebSocket(); },
        fetchData,
        fetchMarketData,
        toggleTheme,
        toggleJson,
        copyJson,
        toggleManualUpdate,
        clearManualForm,
        updateManualRate,
        closeBanner,
        // WebSocket status
        isConnected: () => state.isConnected,
        getConnectionState: () => ({
            connected: state.isConnected,
            reconnectAttempts: state.reconnectAttempts
        }),
        isMarketOpen: () => state.marketOpen
    };

    // ===== START APP =====
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

})();
