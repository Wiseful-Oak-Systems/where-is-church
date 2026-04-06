// Church Finder App - Main JavaScript

const DAY_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

// Global state
let map = null;
let markerLayer = null;
let currentLocation = null;
let selectedChurchId = null;

// ─── API Helper ───────────────────────────────────────────────────────────────

async function api(path, options = {}) {
    const res = await fetch('/api' + path, {
        headers: { 'Content-Type': 'application/json' },
        ...options,
    });
    if (res.status === 401) { window.location.href = '/login'; throw new Error('Session expired'); }
    if (!res.ok) { const err = await res.json(); throw new Error(err.error || 'Request failed'); }
    return res.json();
}

// ─── Toast Notifications ──────────────────────────────────────────────────────

function showToast(message, type = 'info') {
    const container = document.getElementById('toast-container') || createToastContainer();
    const toast = document.createElement('div');
    toast.className = `toast toast--${type}`;
    toast.setAttribute('role', 'alert');

    const icons = { success: '✓', error: '✕', info: 'ℹ', warning: '⚠' };
    toast.innerHTML = `
        <span class="toast__icon">${icons[type] || icons.info}</span>
        <span class="toast__message">${escapeHtml(message)}</span>
        <button class="toast__close" aria-label="Close">&times;</button>
    `;

    toast.querySelector('.toast__close').addEventListener('click', () => dismissToast(toast));
    container.appendChild(toast);

    // Trigger enter animation
    requestAnimationFrame(() => toast.classList.add('toast--visible'));

    // Auto-dismiss after 4 seconds
    setTimeout(() => dismissToast(toast), 4000);
}

function dismissToast(toast) {
    toast.classList.remove('toast--visible');
    toast.classList.add('toast--hiding');
    toast.addEventListener('transitionend', () => toast.remove(), { once: true });
}

function createToastContainer() {
    const container = document.createElement('div');
    container.id = 'toast-container';
    container.className = 'toast-container';
    container.setAttribute('aria-live', 'polite');
    document.body.appendChild(container);
    return container;
}

// ─── Auth Functions ───────────────────────────────────────────────────────────

async function register(name, email, password, denomination) {
    try {
        await api('/auth/register', {
            method: 'POST',
            body: JSON.stringify({ name, email, password, denomination }),
        });
        showToast('Account created! Welcome.', 'success');
        window.location.href = '/';
    } catch (err) {
        showToast(err.message, 'error');
        throw err;
    }
}

async function login(email, password) {
    try {
        await api('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email, password }),
        });
        showToast('Welcome back!', 'success');
        window.location.href = '/';
    } catch (err) {
        showToast(err.message, 'error');
        throw err;
    }
}

async function logout() {
    try {
        await api('/auth/logout', { method: 'POST' });
        window.location.href = '/login';
    } catch (err) {
        showToast(err.message, 'error');
    }
}

async function getMe() {
    try {
        return await api('/auth/me');
    } catch (err) {
        return null;
    }
}

async function updateProfile(name, denomination, latitude, longitude) {
    try {
        const data = await api('/auth/profile', {
            method: 'PUT',
            body: JSON.stringify({ name, denomination, latitude, longitude }),
        });
        showToast('Profile updated.', 'success');
        return data;
    } catch (err) {
        showToast(err.message, 'error');
        throw err;
    }
}

// ─── Map Initialization ───────────────────────────────────────────────────────

// Default center: Brazil (-15.78, -47.93 = Brasília) at zoom 4
function initMap(centerLat = -15.78, centerLng = -47.93, zoom = 4) {
    if (map) return;

    const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;

    map = L.map('map', {
        center: [centerLat, centerLng],
        zoom,
        zoomControl: true,
        tap: false, // prevents 300ms delay on mobile Android
    });

    const lightTiles = L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
        maxZoom: 19,
    });
    const darkTiles = L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OSM</a> &copy; <a href="https://carto.com/">CARTO</a>',
        maxZoom: 19,
    });

    (isDark ? darkTiles : lightTiles).addTo(map);

    // Swap tiles on dark mode change
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
        map.eachLayer(l => { if (l instanceof L.TileLayer) map.removeLayer(l); });
        (e.matches ? darkTiles : lightTiles).addTo(map);
    });

    markerLayer = L.layerGroup().addTo(map);

    // Click on map to set custom location
    map.on('click', (e) => {
        setCustomLocation(e.latlng.lat, e.latlng.lng);
    });
}

function setCustomLocation(lat, lng) {
    currentLocation = { lat, lng };
    updateLocationMarker(lat, lng);
    triggerSearch();
    showToast('Localização definida. Buscando igrejas próximas...', 'info');
}

let locationMarker = null;

function updateLocationMarker(lat, lng) {
    if (locationMarker) {
        locationMarker.setLatLng([lat, lng]);
    } else {
        const icon = L.divIcon({
            className: 'location-marker',
            html: '<div class="location-marker__pulse"></div><div class="location-marker__dot"></div>',
            iconSize: [24, 24],
            iconAnchor: [12, 12],
        });
        locationMarker = L.marker([lat, lng], { icon, zIndexOffset: 1000 })
            .addTo(map)
            .bindTooltip('Your location', { permanent: false });
    }
}

function useMyLocation() {
    const btn = document.getElementById('use-my-location');
    if (btn) { btn.classList.add('btn--loading'); btn.disabled = true; }

    function onLocationFound(lat, lng) {
        currentLocation = { lat, lng };
        if (map) {
            map.setView([lat, lng], 13);
            updateLocationMarker(lat, lng);
        }
        triggerSearch();
        showToast('Localização encontrada!', 'success');
        if (btn) { btn.classList.remove('btn--loading'); btn.disabled = false; }
    }

    function onLocationFailed() {
        // Fallback: try IP-based geolocation
        fallbackIPGeolocation(onLocationFound, () => {
            showToast('Não foi possível detectar a localização. Busque um local ou clique no mapa.', 'warning');
            if (btn) { btn.classList.remove('btn--loading'); btn.disabled = false; }
        });
    }

    if (!navigator.geolocation) {
        onLocationFailed();
        return;
    }

    navigator.geolocation.getCurrentPosition(
        (pos) => onLocationFound(pos.coords.latitude, pos.coords.longitude),
        () => onLocationFailed(),
        { timeout: 8000, maximumAge: 60000 }
    );
}

// IP-based geolocation fallback using free APIs.
// Tries multiple providers for reliability.
async function fallbackIPGeolocation(onSuccess, onError) {
    const providers = [
        {
            url: 'https://ipapi.co/json/',
            parse: (data) => ({ lat: data.latitude, lng: data.longitude, city: data.city }),
        },
        {
            url: 'https://ip-api.com/json/?fields=lat,lon,city',
            parse: (data) => ({ lat: data.lat, lng: data.lon, city: data.city }),
        },
    ];

    for (const provider of providers) {
        try {
            const res = await fetch(provider.url, { signal: AbortSignal.timeout(5000) });
            if (!res.ok) continue;
            const data = await res.json();
            const result = provider.parse(data);
            if (result.lat && result.lng) {
                showToast(`Localização aproximada: ${result.city || 'detectada via IP'}`, 'info');
                onSuccess(result.lat, result.lng);
                return;
            }
        } catch {
            // Try next provider
        }
    }
    onError();
}

// ─── Church Search ────────────────────────────────────────────────────────────

async function searchChurches(lat, lng, radius, denomination) {
    clearMarkers();
    showSkeletonCards();

    const params = new URLSearchParams({ lat, lng, radius });
    if (denomination && denomination !== 'all') {
        params.set('denomination', denomination);
    }

    try {
        const churches = await api(`/churches/search?${params}`);
        const list = document.getElementById('church-list');
        const countEl = document.getElementById('results-count');

        if (!churches || churches.length === 0) {
            if (list) list.innerHTML = '<p class="church-list-empty">Nenhuma igreja encontrada nesta área. Tente aumentar o raio ou mudar a denominação.</p>';
            if (countEl) countEl.textContent = '0 results';
            return [];
        }
        churches.forEach((church) => placeChurchMarker(church));
        if (countEl) countEl.textContent = `${churches.length} result${churches.length !== 1 ? 's' : ''}`;

        // Render church cards in sidebar
        if (list) {
            list.innerHTML = churches.map(ch => `
                <article class="church-card" role="listitem" onclick="window.location='/church/${ch.id}'">
                    <div class="church-card-body">
                        <h3 class="church-card-name">${escapeHtml(ch.name)}</h3>
                        <span class="denomination-badge denomination-badge--${(ch.denomination || 'other').toLowerCase()}">${escapeHtml(ch.denomination)}</span>
                        <p class="church-card-address">${escapeHtml(ch.address || '')}</p>
                        <div class="church-card-meta">
                            <span class="church-card-distance">${ch.distance != null ? ch.distance.toFixed(1) + ' km' : ''}</span>
                        </div>
                    </div>
                    <div class="church-card-actions">
                        <a href="/church/${ch.id}" class="btn btn-sm btn-outline">View</a>
                    </div>
                </article>
            `).join('');
        }

        return churches;
    } catch (err) {
        const list = document.getElementById('church-list');
        if (list) list.innerHTML = '<p class="church-list-empty">Falha na busca. Tente novamente.</p>';
        showToast('Search failed: ' + err.message, 'error');
        return [];
    }
}

function triggerSearch() {
    if (!currentLocation) return;

    const radius = getRadiusValue();
    const denomination = getDenominationFilter();
    searchChurches(currentLocation.lat, currentLocation.lng, radius, denomination);
}

function getRadiusValue() {
    const slider = document.getElementById('radius-slider');
    return slider ? parseInt(slider.value, 10) : 10;
}

function getDenominationFilter() {
    const select = document.getElementById('denomination-filter');
    return select ? select.value : 'all';
}

// ─── Church Markers ───────────────────────────────────────────────────────────

function clearMarkers() {
    if (markerLayer) markerLayer.clearLayers();
}

function placeChurchMarker(church) {
    if (!markerLayer) return;

    const denomClass = 'marker-' + (church.denomination || 'default').toLowerCase();
    const icon = L.divIcon({
        className: '',
        html: `<div class="church-marker ${denomClass}" title="${escapeHtml(church.name)}"><span>&#9962;</span></div>`,
        iconSize: [28, 28],
        iconAnchor: [14, 28],
        popupAnchor: [0, -30],
    });

    const distKm = church.distance != null ? church.distance : null;
    const distanceText = distKm != null ? `${distKm.toFixed(1)} km away` : '';
    const massCount = (church.schedules || []).length;
    const directionsUrl = `https://www.openstreetmap.org/directions?from=&to=${church.latitude},${church.longitude}`;

    const popupContent = `
        <div class="church-popup">
            <h3 class="popup-title">${escapeHtml(church.name)}</h3>
            <span class="denomination-badge denomination-badge--${(church.denomination || 'other').toLowerCase()}">${escapeHtml(church.denomination || 'Unknown')}</span>
            ${distanceText ? `<p class="popup-distance">${escapeHtml(distanceText)}</p>` : ''}
            ${church.address ? `<p class="popup-address">${escapeHtml(church.address)}</p>` : ''}
            ${massCount > 0 ? `<p class="popup-address">${massCount} scheduled service${massCount !== 1 ? 's' : ''}</p>` : ''}
            <a href="${directionsUrl}" target="_blank" rel="noopener" class="directions-link">&#x2794; Get directions</a>
            <div style="margin-top:.5rem;display:flex;gap:.3rem">
                <a href="/church/${church.id}" class="btn btn-sm btn-primary popup-link">View</a>
                <button class="btn btn-sm btn-outline" onclick="checkIn('${church.id}')">Check In</button>
            </div>
        </div>
    `;

    const marker = L.marker([church.latitude, church.longitude], {
            icon,
            keyboard: true,
            alt: `${church.name} - ${church.denomination}${distanceText ? ', ' + distanceText : ''}`,
        })
        .bindPopup(popupContent, { maxWidth: 280 })
        .on('popupopen', () => announceToScreenReader(`${church.name}, ${church.denomination}${distanceText ? ', ' + distanceText : ''}`))
        .addTo(markerLayer);

    marker.on('click', () => {
        selectedChurchId = church.id;
    });
}

// ─── Church Detail ────────────────────────────────────────────────────────────

async function loadChurchDetail(id) {
    selectedChurchId = id;
    const panel = document.getElementById('church-detail-panel');
    if (panel) {
        panel.innerHTML = '<div class="loading-spinner" aria-label="Loading..."></div>';
        openSidebar();
    }

    try {
        const church = await api(`/churches/${id}`);
        renderChurchDetail(church);
    } catch (err) {
        showToast('Could not load church details: ' + err.message, 'error');
        if (panel) panel.innerHTML = '<p class="error-text">Failed to load church details.</p>';
    }
}

function renderChurchDetail(church) {
    const panel = document.getElementById('church-detail-panel');
    if (!panel) return;

    const schedules = Array.isArray(church.schedules) && church.schedules.length > 0
        ? church.schedules.map((s) => `
            <li class="schedule-item">
                <strong>${DAY_NAMES[s.day_of_week] || 'Unknown'}</strong>
                ${escapeHtml(s.start_time)}
                ${s.language ? `<span class="schedule-item__lang">(${escapeHtml(s.language)})</span>` : ''}
                ${s.notes ? `<span class="schedule-item__notes"> — ${escapeHtml(s.notes)}</span>` : ''}
            </li>
        `).join('')
        : '<li class="schedule-item schedule-item--empty">No schedule listed.</li>';

    panel.innerHTML = `
        <div class="church-detail">
            <button class="church-detail__close btn btn--ghost" onclick="closeSidebar()" aria-label="Close">&times;</button>
            <h2 class="church-detail__name">${escapeHtml(church.name)}</h2>
            <p class="church-detail__denomination badge">${escapeHtml(church.denomination || 'Unknown')}</p>

            ${church.address ? `
                <div class="church-detail__section">
                    <h4>Address</h4>
                    <p>${escapeHtml(church.address)}</p>
                </div>
            ` : ''}

            ${church.phone || church.website ? `
                <div class="church-detail__section">
                    <h4>Contact</h4>
                    ${church.phone ? `<p><a href="tel:${escapeHtml(church.phone)}">${escapeHtml(church.phone)}</a></p>` : ''}
                    ${church.website ? `<p><a href="${escapeHtml(church.website)}" target="_blank" rel="noopener noreferrer">${escapeHtml(church.website)}</a></p>` : ''}
                </div>
            ` : ''}

            ${church.description ? `
                <div class="church-detail__section">
                    <h4>About</h4>
                    <p>${escapeHtml(church.description)}</p>
                </div>
            ` : ''}

            <div class="church-detail__section">
                <h4>Service Schedule</h4>
                <ul class="schedule-list">${schedules}</ul>
            </div>

            <div class="church-detail__actions">
                <button class="btn btn--primary" onclick="checkIn('${church.id}')">
                    Check In Here
                </button>
                <button class="btn btn--secondary" onclick="openSuggestionModal('${church.id}')">
                    Suggest a Change
                </button>
            </div>
        </div>
    `;
}

// ─── Check-In ─────────────────────────────────────────────────────────────────

async function checkIn(churchId, notes = '') {
    try {
        await api('/checkins', {
            method: 'POST',
            body: JSON.stringify({ church_id: churchId, notes }),
        });
        showCheckInAnimation();
        showToast('You have checked in! God bless you.', 'success');
    } catch (err) {
        showToast('Check-in failed: ' + err.message, 'error');
    }
}

function showCheckInAnimation() {
    const overlay = document.createElement('div');
    overlay.className = 'checkin-animation';
    overlay.innerHTML = `
        <div class="checkin-animation__content">
            <div class="checkin-animation__icon">&#9962;</div>
            <p class="checkin-animation__text">Checked In!</p>
        </div>
    `;
    document.body.appendChild(overlay);

    requestAnimationFrame(() => overlay.classList.add('checkin-animation--visible'));

    setTimeout(() => {
        overlay.classList.remove('checkin-animation--visible');
        overlay.addEventListener('transitionend', () => overlay.remove(), { once: true });
    }, 1800);
}

async function getMyCheckins() {
    try {
        return await api('/checkins/mine');
    } catch (err) {
        showToast('Could not load check-ins: ' + err.message, 'error');
        return [];
    }
}

async function getCheckinStats() {
    try {
        return await api('/checkins/stats');
    } catch (err) {
        showToast('Could not load stats.', 'error');
        return null;
    }
}

// ─── Suggestions ──────────────────────────────────────────────────────────────

function openSuggestionModal(churchId) {
    selectedChurchId = churchId;
    const modal = document.getElementById('suggestion-modal');
    if (modal) {
        modal.removeAttribute('hidden');
        modal.setAttribute('aria-modal', 'true');
        const firstInput = modal.querySelector('select, input, textarea');
        if (firstInput) firstInput.focus();
    }
}

function closeSuggestionModal() {
    const modal = document.getElementById('suggestion-modal');
    if (modal) modal.setAttribute('hidden', '');
}

async function submitSuggestion(churchId, type, content) {
    try {
        await api('/suggestions', {
            method: 'POST',
            body: JSON.stringify({
                church_id: churchId || selectedChurchId,
                type,
                content,
            }),
        });
        showToast('Suggestion submitted. Thank you!', 'success');
        closeSuggestionModal();
    } catch (err) {
        showToast('Could not submit suggestion: ' + err.message, 'error');
    }
}

async function getMySuggestions() {
    try {
        return await api('/suggestions/mine');
    } catch (err) {
        showToast('Could not load suggestions.', 'error');
        return [];
    }
}

// ─── Sidebar Toggle ───────────────────────────────────────────────────────────

function openSidebar() {
    const sidebar = document.getElementById('sidebar');
    if (sidebar) {
        sidebar.classList.add('sidebar--open');
        sidebar.removeAttribute('hidden');
    }
}

function closeSidebar() {
    const sidebar = document.getElementById('sidebar');
    if (sidebar) {
        sidebar.classList.remove('sidebar--open');
    }
}

function toggleSidebar() {
    const sidebar = document.getElementById('sidebar');
    if (sidebar) {
        sidebar.classList.toggle('sidebar--open');
    }
}

// ─── UI Control Helpers ───────────────────────────────────────────────────────

function initRadiusSlider() {
    const slider = document.getElementById('radius-slider');
    const label = document.getElementById('radius-label');
    if (!slider) return;

    slider.addEventListener('input', () => {
        if (label) label.textContent = slider.value;
    });

    slider.addEventListener('change', () => {
        triggerSearch();
    });

    if (label) label.textContent = slider.value;
}

function initDenominationFilter() {
    const select = document.getElementById('denomination-filter');
    if (!select) return;

    // Set default from user's denomination (stored in body data attribute)
    const userDenom = document.body.dataset.denomination;
    if (userDenom && userDenom !== '') {
        const option = select.querySelector(`option[value="${userDenom}"]`);
        if (option) {
            select.value = userDenom;
        }
    }

    select.addEventListener('change', () => triggerSearch());
}

function initSuggestionForm() {
    const form = document.getElementById('suggestion-form');
    if (!form) return;

    const typeSelect = document.getElementById('suggestion-type');
    const proposalFields = document.getElementById('proposal-fields');
    const churchSearchGroup = document.getElementById('church-search-group');
    const contentField = document.getElementById('suggestion-content');

    // Show/hide fields based on suggestion type
    function updateFormFields() {
        const type = typeSelect?.value;
        if (proposalFields) proposalFields.hidden = type !== 'new_church';
        if (churchSearchGroup) churchSearchGroup.hidden = (type === 'new_church' || type === 'general');

        // Update placeholder based on type
        if (contentField) {
            const placeholders = {
                'new_church': 'Horários de missa, características especiais, como chegar...',
                'edit_church': 'O que precisa ser corrigido? (endereço, telefone, nome...)',
                'schedule': 'Quais são os horários corretos de missa?',
                'general': 'Seu feedback ou sugestão...',
            };
            contentField.placeholder = placeholders[type] || 'Details...';
        }
    }

    if (typeSelect) {
        typeSelect.addEventListener('change', updateFormFields);
        updateFormFields();
    }

    // Church search within the modal (for edit/schedule types)
    const churchSearch = document.getElementById('suggestion-church-search');
    const churchResults = document.getElementById('church-search-results');
    let churchSearchTimeout = null;

    if (churchSearch) {
        churchSearch.addEventListener('input', () => {
            clearTimeout(churchSearchTimeout);
            const q = churchSearch.value.trim();
            if (q.length < 2) { if (churchResults) churchResults.hidden = true; return; }
            churchSearchTimeout = setTimeout(async () => {
                try {
                    const churches = await api('/churches?limit=5&denomination=All');
                    const filtered = churches.filter(c =>
                        c.name.toLowerCase().includes(q.toLowerCase()) ||
                        (c.address && c.address.toLowerCase().includes(q.toLowerCase()))
                    );
                    if (!filtered.length) { churchResults.hidden = true; return; }
                    churchResults.innerHTML = filtered.map(c =>
                        `<li class="search-result-item" role="option" data-id="${c.id}">${escapeHtml(c.name)} <small class="text-muted">${escapeHtml(c.address || '')}</small></li>`
                    ).join('');
                    churchResults.hidden = false;
                    churchResults.querySelectorAll('.search-result-item').forEach(item => {
                        item.addEventListener('click', () => {
                            document.getElementById('suggestion-church-id').value = item.dataset.id;
                            const selected = document.getElementById('selected-church-name');
                            if (selected) { selected.textContent = '✓ Selected: ' + item.textContent; selected.hidden = false; }
                            churchSearch.value = '';
                            churchResults.hidden = true;
                        });
                    });
                } catch { churchResults.hidden = true; }
            }, 300);
        });
    }

    // Auto-geocode when address is filled (so users don't need to enter coordinates)
    const proposalAddress = document.getElementById('proposal-address');
    if (proposalAddress) {
        let geoTimeout = null;
        proposalAddress.addEventListener('blur', () => {
            const addr = proposalAddress.value.trim();
            const latField = document.getElementById('proposal-lat');
            const lngField = document.getElementById('proposal-lng');
            if (addr.length > 5 && latField && (!latField.value || latField.value === '0')) {
                clearTimeout(geoTimeout);
                geoTimeout = setTimeout(async () => {
                    try {
                        const res = await fetch(`https://nominatim.openstreetmap.org/search?q=${encodeURIComponent(addr + ', Brasil')}&format=json&limit=1`);
                        const results = await res.json();
                        if (results.length > 0) {
                            latField.value = parseFloat(results[0].lat).toFixed(6);
                            lngField.value = parseFloat(results[0].lon).toFixed(6);
                            showToast('Localização detectada automaticamente pelo endereço!', 'success');
                        }
                    } catch { /* ignore geocoding errors */ }
                }, 500);
            }
        });
    }

    // Pick location from map for proposals
    const pickBtn = document.getElementById('proposal-pick-location');
    if (pickBtn) {
        pickBtn.addEventListener('click', () => {
            closeSuggestionModal();
            showToast('Clique no mapa para definir a localização da igreja.', 'info');
            if (map) map.getContainer().style.cursor = 'crosshair';
            const handler = (e) => {
                document.getElementById('proposal-lat').value = e.latlng.lat.toFixed(6);
                document.getElementById('proposal-lng').value = e.latlng.lng.toFixed(6);
                map.getContainer().style.cursor = '';
                map.off('click', handler);
                showToast('Localização definida! Reabrindo o formulário...', 'success');
                setTimeout(() => openSuggestionModal(), 500);
            };
            if (map) map.on('click', handler);
        });
    }

    // Form submission
    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const type = typeSelect?.value;
        const content = contentField?.value;
        if (!type || !content) {
            showToast('Preencha todos os campos obrigatórios.', 'warning');
            return;
        }

        const body = { type, content };

        // For new_church, include the structured proposal
        if (type === 'new_church') {
            const name = document.getElementById('proposal-name')?.value;
            const address = document.getElementById('proposal-address')?.value;
            if (!name) { showToast('Informe o nome da igreja.', 'warning'); return; }
            body.proposal = {
                name,
                address: address || '',
                denomination: document.getElementById('proposal-denomination')?.value || 'Catholic',
                phone: document.getElementById('proposal-phone')?.value || '',
                latitude: parseFloat(document.getElementById('proposal-lat')?.value) || 0,
                longitude: parseFloat(document.getElementById('proposal-lng')?.value) || 0,
            };
        }

        // For edit/schedule, include church_id
        const churchId = document.getElementById('suggestion-church-id')?.value;
        if (churchId) body.church_id = parseInt(churchId);

        try {
            await api('/suggestions', { method: 'POST', body: JSON.stringify(body) });
            showToast('Sugestão enviada! Obrigado pela contribuição.', 'success');
            closeSuggestionModal();
            form.reset();
            updateFormFields();
        } catch (err) {
            showToast(err.message, 'error');
        }
    });

    const cancelBtn = document.getElementById('suggestion-cancel');
    if (cancelBtn) cancelBtn.addEventListener('click', closeSuggestionModal);
    const closeBtn = document.getElementById('suggestion-close');
    if (closeBtn) closeBtn.addEventListener('click', closeSuggestionModal);
    const backdrop = document.getElementById('suggestion-backdrop');
    if (backdrop) backdrop.addEventListener('click', closeSuggestionModal);
    const fab = document.getElementById('suggest-fab');
    if (fab) fab.addEventListener('click', () => openSuggestionModal());
}

function initAuthForms() {
    // Note: login.html and register.html have their own inline <script> handlers.
    // This function only wires the logout button and navbar toggle, which are
    // shared across all authenticated pages.

    // Wire logout button (present on all authenticated pages)
    const logoutBtn = document.getElementById('logout-btn');
    if (logoutBtn) {
        logoutBtn.addEventListener('click', (e) => {
            e.preventDefault();
            logout();
        });
    }
}

function initLocationBtn() {
    const btn = document.getElementById('use-my-location');
    if (btn) btn.addEventListener('click', useMyLocation);

    // "Set location on map" toggle (Bug B1)
    const setBtn = document.getElementById('set-location-btn');
    if (setBtn) {
        let pickMode = false;
        setBtn.addEventListener('click', () => {
            pickMode = !pickMode;
            setBtn.setAttribute('aria-pressed', pickMode ? 'true' : 'false');
            setBtn.classList.toggle('btn-primary', pickMode);
            setBtn.classList.toggle('btn-outline', !pickMode);
            if (pickMode) {
                showToast('Clique no mapa para definir sua localização.', 'info');
                if (map) map.getContainer().style.cursor = 'crosshair';
            } else {
                if (map) map.getContainer().style.cursor = '';
            }
        });
    }
}

function initSidebarToggle() {
    const toggleBtn = document.getElementById('sidebar-toggle');
    if (toggleBtn) toggleBtn.addEventListener('click', toggleSidebarWithOverlay);

    const overlay = document.getElementById('sidebar-overlay');
    if (overlay) overlay.addEventListener('click', () => {
        closeSidebar();
        overlay.classList.remove('open');
    });

    // Mobile nav toggle (fix B14: update aria-expanded)
    const navToggle = document.getElementById('nav-toggle');
    if (navToggle) {
        navToggle.addEventListener('click', () => {
            const links = document.querySelector('.navbar-links');
            if (links) {
                links.classList.toggle('open');
                const expanded = links.classList.contains('open');
                navToggle.setAttribute('aria-expanded', expanded ? 'true' : 'false');
            }
        });
    }
}

function toggleSidebarWithOverlay() {
    toggleSidebar();
    const overlay = document.getElementById('sidebar-overlay');
    const sidebar = document.getElementById('sidebar');
    if (overlay && sidebar) {
        overlay.classList.toggle('open', sidebar.classList.contains('sidebar--open'));
    }
}

// ─── Utility ──────────────────────────────────────────────────────────────────

function escapeHtml(str) {
    if (str == null) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

// ─── Screen Reader Announcements ─────────────────────────────────────────────

function announceToScreenReader(message) {
    const el = document.getElementById('map-announcer');
    if (el) { el.textContent = message; }
}

// ─── Skeleton Loading ────────────────────────────────────────────────────────

function showSkeletonCards() {
    const list = document.getElementById('church-list');
    if (!list) return;
    list.innerHTML = Array(3).fill('<div class="skeleton skeleton-card" aria-hidden="true"></div>').join('');
}

// ─── Address / Place Search (Nominatim) ──────────────────────────────────────

let searchTimeout = null;

function initAddressSearch() {
    const input = document.getElementById('address-search');
    const btn = document.getElementById('address-search-btn');
    const resultsList = document.getElementById('address-results');
    if (!input) return;

    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') { e.preventDefault(); geocodeAddress(input.value); }
    });
    if (btn) btn.addEventListener('click', () => geocodeAddress(input.value));

    // Live suggestions with debounce
    input.addEventListener('input', () => {
        clearTimeout(searchTimeout);
        const q = input.value.trim();
        if (q.length < 3) { if (resultsList) resultsList.hidden = true; return; }
        searchTimeout = setTimeout(() => geocodeSuggestions(q), 300);
    });

    // Close results on outside click
    document.addEventListener('click', (e) => {
        if (resultsList && !resultsList.contains(e.target) && e.target !== input) {
            resultsList.hidden = true;
        }
    });
}

async function geocodeSuggestions(query) {
    const resultsList = document.getElementById('address-results');
    if (!resultsList) return;
    try {
        const res = await fetch(`https://nominatim.openstreetmap.org/search?q=${encodeURIComponent(query)}&format=json&limit=5`, {
            headers: { 'Accept': 'application/json' },
        });
        const results = await res.json();
        if (!results.length) { resultsList.hidden = true; return; }

        resultsList.innerHTML = results.map((r, i) =>
            `<li class="search-result-item" role="option" tabindex="0" data-lat="${r.lat}" data-lon="${r.lon}">${escapeHtml(r.display_name)}</li>`
        ).join('');
        resultsList.hidden = false;

        resultsList.querySelectorAll('.search-result-item').forEach(item => {
            const handler = () => {
                const lat = parseFloat(item.dataset.lat);
                const lon = parseFloat(item.dataset.lon);
                document.getElementById('address-search').value = item.textContent;
                resultsList.hidden = true;
                setCustomLocation(lat, lon);
                map.setView([lat, lon], 13);
            };
            item.addEventListener('click', handler);
            item.addEventListener('keydown', (e) => { if (e.key === 'Enter') handler(); });
        });
    } catch (err) {
        resultsList.hidden = true;
    }
}

async function geocodeAddress(query) {
    if (!query || query.trim().length < 2) return;
    const resultsList = document.getElementById('address-results');
    if (resultsList) resultsList.hidden = true;

    try {
        const res = await fetch(`https://nominatim.openstreetmap.org/search?q=${encodeURIComponent(query)}&format=json&limit=1`);
        const results = await res.json();
        if (!results.length) {
            showToast('Location not found. Try a different search.', 'warning');
            return;
        }
        const { lat, lon, display_name } = results[0];
        setCustomLocation(parseFloat(lat), parseFloat(lon));
        map.setView([parseFloat(lat), parseFloat(lon)], 13);
        showToast(`Found: ${display_name.split(',').slice(0, 2).join(',')}`, 'success');
    } catch (err) {
        showToast('Search failed. Please try again.', 'error');
    }
}

// ─── Bottom Sheet (Mobile) ───────────────────────────────────────────────────

function initBottomSheet() {
    if (window.innerWidth > 768) return;

    const sidebar = document.getElementById('sidebar');
    if (!sidebar) return;
    sidebar.classList.add('bottom-sheet');

    const handle = sidebar.querySelector('.bottom-sheet-handle');
    if (!handle) return;

    let startY = 0, startTranslate = 0, currentTranslate = 0, isDragging = false;
    const maxH = sidebar.offsetHeight || window.innerHeight * 0.85;
    const collapsed = maxH - 110;
    const half = maxH * 0.35;

    handle.addEventListener('pointerdown', (e) => {
        isDragging = true;
        startY = e.clientY;
        const transform = getComputedStyle(sidebar).transform;
        const matrix = new DOMMatrixReadOnly(transform);
        startTranslate = matrix.m42;
        sidebar.style.transition = 'none';
        handle.setPointerCapture(e.pointerId);
    });

    handle.addEventListener('pointermove', (e) => {
        if (!isDragging) return;
        const dy = e.clientY - startY;
        currentTranslate = Math.max(0, Math.min(maxH, startTranslate + dy));
        sidebar.style.transform = `translateY(${currentTranslate}px)`;
    });

    handle.addEventListener('pointerup', () => {
        if (!isDragging) return;
        isDragging = false;
        sidebar.style.transition = '';

        // Snap to nearest state
        if (currentTranslate < half * 0.5) {
            sidebar.classList.add('full');
            sidebar.classList.remove('half');
        } else if (currentTranslate < collapsed * 0.7) {
            sidebar.classList.add('half');
            sidebar.classList.remove('full');
        } else {
            sidebar.classList.remove('half', 'full');
        }
        sidebar.style.transform = '';
    });

    // Tap handle to toggle half/collapsed
    handle.addEventListener('click', () => {
        if (sidebar.classList.contains('half') || sidebar.classList.contains('full')) {
            sidebar.classList.remove('half', 'full');
        } else {
            sidebar.classList.add('half');
        }
    });
}

// ─── App Initialization ───────────────────────────────────────────────────────

async function init() {
    // Initialize UI controls regardless of map (these work without Leaflet)
    initRadiusSlider();
    initDenominationFilter();
    initLocationBtn();
    initSidebarToggle();
    initSuggestionForm();
    initAddressSearch();
    initBottomSheet();

    // Initialize map if the map element exists AND Leaflet loaded
    const mapEl = document.getElementById('map');
    if (mapEl && typeof L !== 'undefined') {
        try {
            initMap();
            // Auto-detect location on first load
            useMyLocation();
        } catch (err) {
            console.error('Map initialization failed:', err);
            showToast('Falha ao carregar o mapa. Atualize a página.', 'error');
        }
    } else if (mapEl) {
        console.warn('Leaflet library not loaded. Map disabled.');
        mapEl.innerHTML = '<div style="display:flex;align-items:center;justify-content:center;height:100%;color:var(--color-gray-400);text-align:center;padding:2rem"><p>Carregando mapa...<br>Se persistir, verifique sua conexão e atualize a página.</p></div>';
    }

    // Initialize auth forms
    initAuthForms();

    // Church detail page
    const churchId = document.body.dataset.churchId;
    if (churchId) {
        initChurchDetailPage(churchId);
    }

    // Profile page
    const profileSection = document.getElementById('profile-section');
    if (profileSection) {
        initProfilePage();
    }

    // Admin page
    const adminSection = document.getElementById('admin-section');
    if (adminSection) {
        initAdminPage();
    }
}

// ─── Church Detail Page ──────────────────────────────────────────────────────

async function initChurchDetailPage(churchId) {
    const loading = document.getElementById('church-loading');
    const error = document.getElementById('church-error');
    const content = document.getElementById('church-content');

    try {
        const church = await api('/churches/' + churchId);
        if (loading) loading.hidden = true;
        if (content) content.hidden = false;

        // Populate fields
        setText('church-name', church.name);
        setText('church-denomination', church.denomination);
        setText('church-address', church.address);

        const verified = document.getElementById('church-verified');
        if (verified && church.verified) verified.hidden = false;

        if (church.phone) {
            show('phone-row');
            const phoneEl = document.getElementById('church-phone');
            if (phoneEl) { phoneEl.textContent = church.phone; phoneEl.href = 'tel:' + church.phone; }
        }
        if (church.website) {
            show('website-row');
            const webEl = document.getElementById('church-website');
            if (webEl) { webEl.textContent = church.website; webEl.href = church.website; }
        }
        if (church.description) {
            show('description-row');
            setText('church-description', church.description);
        }

        // Init mini map
        const mapEl = document.getElementById('church-map');
        if (mapEl && church.latitude && church.longitude) {
            const miniMap = L.map(mapEl).setView([church.latitude, church.longitude], 15);
            L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
                attribution: '&copy; OpenStreetMap contributors'
            }).addTo(miniMap);
            L.marker([church.latitude, church.longitude]).addTo(miniMap);
        }

        // Schedule
        renderSchedule(church.schedules || []);

        // Check-ins
        loadChurchCheckins(churchId);

        // Check-in buttons
        const checkinBtns = document.querySelectorAll('#checkin-btn, #checkin-btn-2');
        checkinBtns.forEach(btn => {
            btn.addEventListener('click', () => doCheckin(churchId));
        });

        // Add schedule form
        const schedForm = document.getElementById('add-schedule-form');
        if (schedForm) {
            schedForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                try {
                    await api('/churches/' + churchId + '/schedules', {
                        method: 'POST',
                        body: JSON.stringify({
                            day_of_week: parseInt(document.getElementById('sched-day').value),
                            start_time: document.getElementById('sched-time').value,
                            language: document.getElementById('sched-lang').value || 'English',
                            notes: document.getElementById('sched-notes').value,
                        }),
                    });
                    showToast('Schedule added!', 'success');
                    const updated = await api('/churches/' + churchId);
                    renderSchedule(updated.schedules || []);
                    schedForm.reset();
                } catch (err) {
                    showFormError('add-sched-error', err.message);
                }
            });
        }

        // Edit church form
        const editForm = document.getElementById('edit-church-form');
        if (editForm) {
            // Pre-fill
            setVal('edit-name', church.name);
            setVal('edit-address', church.address);
            setVal('edit-phone', church.phone);
            setVal('edit-website', church.website);
            setVal('edit-description', church.description);
            const denomSelect = document.getElementById('edit-denomination');
            if (denomSelect) denomSelect.value = church.denomination;
            const verifiedCb = document.getElementById('edit-verified');
            if (verifiedCb) verifiedCb.checked = church.verified;

            editForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                try {
                    await api('/churches/' + churchId, {
                        method: 'PUT',
                        body: JSON.stringify({
                            name: document.getElementById('edit-name').value,
                            denomination: document.getElementById('edit-denomination').value,
                            address: document.getElementById('edit-address').value,
                            phone: document.getElementById('edit-phone').value,
                            website: document.getElementById('edit-website').value,
                            description: document.getElementById('edit-description').value,
                            verified: document.getElementById('edit-verified')?.checked || false,
                        }),
                    });
                    showFormSuccess('edit-church-success', 'Church updated!');
                    showToast('Church updated!', 'success');
                } catch (err) {
                    showFormError('edit-church-error', err.message);
                }
            });
        }

        // Suggestion form for this church
        const suggForm = document.getElementById('church-suggestion-form');
        if (suggForm) {
            suggForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                try {
                    await api('/suggestions', {
                        method: 'POST',
                        body: JSON.stringify({
                            church_id: parseInt(churchId),
                            type: document.getElementById('church-sugg-type').value,
                            content: document.getElementById('church-sugg-content').value,
                        }),
                    });
                    showFormSuccess('church-sugg-success', 'Suggestion submitted! Thank you.');
                    suggForm.reset();
                } catch (err) {
                    showFormError('church-sugg-error', err.message);
                }
            });
        }

    } catch (err) {
        if (loading) loading.hidden = true;
        if (error) error.hidden = false;
    }
}

function renderSchedule(schedules) {
    const tbody = document.getElementById('schedule-tbody');
    const empty = document.getElementById('schedule-empty');
    if (!tbody) return;

    if (!schedules.length) {
        if (empty) empty.hidden = false;
        tbody.innerHTML = '';
        return;
    }
    if (empty) empty.hidden = true;

    const sorted = [...schedules].sort((a, b) => a.day_of_week - b.day_of_week || a.start_time.localeCompare(b.start_time));
    tbody.innerHTML = sorted.map(s => `
        <tr>
            <td class="schedule-day">${DAY_NAMES[s.day_of_week]}</td>
            <td>${escapeHtml(s.start_time)}</td>
            <td>${escapeHtml(s.language)}</td>
            <td>${escapeHtml(s.notes || '—')}</td>
        </tr>
    `).join('');
}

async function loadChurchCheckins(churchId) {
    try {
        const checkins = await api('/churches/' + churchId + '/checkins');
        const list = document.getElementById('checkin-list');
        const empty = document.getElementById('checkin-list-empty');
        if (!list) return;

        if (!checkins.length) {
            if (empty) empty.hidden = false;
            return;
        }
        if (empty) empty.hidden = true;

        list.innerHTML = checkins.map(c => {
            const name = c.user?.name || 'Anonymous';
            const initial = name.charAt(0).toUpperCase();
            const time = new Date(c.created_at).toLocaleDateString();
            return `
                <li class="checkin-item">
                    <div class="checkin-avatar">${initial}</div>
                    <div class="checkin-details">
                        <span class="checkin-name">${escapeHtml(name)}</span>
                        <span class="checkin-time">${time}</span>
                    </div>
                </li>`;
        }).join('');
    } catch (err) {
        // silently fail
    }
}

async function doCheckin(churchId) {
    try {
        await api('/checkins', {
            method: 'POST',
            body: JSON.stringify({ church_id: parseInt(churchId) }),
        });
        showToast('Checked in! God bless.', 'success');
        const msg = document.getElementById('checkin-msg');
        if (msg) { msg.textContent = 'Checked in successfully!'; msg.hidden = false; }
        loadChurchCheckins(churchId);
    } catch (err) {
        const errEl = document.getElementById('checkin-error');
        if (errEl) { errEl.textContent = err.message; errEl.hidden = false; }
        showToast(err.message, 'error');
    }
}

// ─── Profile Page ────────────────────────────────────────────────────────────

async function initProfilePage() {
    const loading = document.getElementById('profile-loading');
    const content = document.getElementById('profile-content');

    try {
        const [user, stats, checkins, suggestions] = await Promise.all([
            api('/auth/me'),
            api('/checkins/stats'),
            api('/checkins/mine'),
            api('/suggestions/mine'),
        ]);

        if (loading) loading.hidden = true;
        if (content) content.hidden = false;

        // Profile info
        setText('profile-name', user.name);
        setText('profile-email', user.email);
        setText('profile-denomination', user.denomination);
        setText('profile-role', user.role);
        setText('profile-avatar-initials', user.name.charAt(0).toUpperCase());

        // Stats
        setText('stat-total', stats.total_checkins);
        setText('stat-recent', stats.recent_30_days);

        // Top churches
        if (stats.top_churches?.length) {
            show('top-churches-section');
            const topList = document.getElementById('top-churches-list');
            if (topList) {
                topList.innerHTML = stats.top_churches.map(c =>
                    `<li>${escapeHtml(c.church_name)} (${c.visits} visits)</li>`
                ).join('');
            }
        }

        // Edit form
        const editForm = document.getElementById('edit-profile-form');
        if (editForm) {
            setVal('edit-name', user.name);
            const denomSelect = document.getElementById('edit-denomination');
            if (denomSelect) denomSelect.value = user.denomination;

            editForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                try {
                    await api('/auth/profile', {
                        method: 'PUT',
                        body: JSON.stringify({
                            name: document.getElementById('edit-name').value,
                            denomination: document.getElementById('edit-denomination').value,
                        }),
                    });
                    showFormSuccess('edit-profile-success', 'Profile updated!');
                    showToast('Profile updated!', 'success');
                } catch (err) {
                    showFormError('edit-profile-error', err.message);
                }
            });
        }

        // Recent check-ins
        const list = document.getElementById('recent-checkins');
        const checkinsEmpty = document.getElementById('checkins-empty');
        if (list) {
            if (checkins.length) {
                list.innerHTML = checkins.slice(0, 10).map(c => `
                    <li class="checkin-item">
                        <div class="checkin-details">
                            <a href="/church/${c.church_id}" class="checkin-name">${escapeHtml(c.church?.name || 'Church #' + c.church_id)}</a>
                            <span class="checkin-time">${new Date(c.created_at).toLocaleDateString()}</span>
                        </div>
                    </li>`).join('');
            } else if (checkinsEmpty) {
                checkinsEmpty.hidden = false;
            }
        }

        // Suggestions
        const suggList = document.getElementById('my-suggestions');
        const suggsEmpty = document.getElementById('suggestions-empty');
        if (suggList) {
            if (suggestions.length) {
                suggList.innerHTML = suggestions.map(s => `
                    <li class="suggestion-item">
                        <div class="suggestion-item-header">
                            <span class="suggestion-type">${escapeHtml(s.type)}</span>
                            <span class="status-badge status-badge--${s.status}">${s.status}</span>
                        </div>
                        <p class="suggestion-content">${escapeHtml(s.content)}</p>
                    </li>`).join('');
            } else if (suggsEmpty) {
                suggsEmpty.hidden = false;
            }
        }
    } catch (err) {
        if (loading) loading.hidden = true;
        showToast('Failed to load profile', 'error');
    }
}

// ─── Admin Page ──────────────────────────────────────────────────────────────

async function initAdminPage() {
    // Tab switching
    const tabs = document.querySelectorAll('.tab[role="tab"]');
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            tabs.forEach(t => {
                t.classList.remove('tab--active');
                t.setAttribute('aria-selected', 'false');
                const panel = document.getElementById(t.getAttribute('aria-controls'));
                if (panel) panel.hidden = true;
            });
            tab.classList.add('tab--active');
            tab.setAttribute('aria-selected', 'true');
            const panel = document.getElementById(tab.getAttribute('aria-controls'));
            if (panel) panel.hidden = false;
        });
    });

    // Refresh buttons
    const refreshUsers = document.getElementById('refresh-users-btn');
    if (refreshUsers) refreshUsers.addEventListener('click', loadAdminUsers);
    const refreshChurches = document.getElementById('refresh-churches-btn');
    if (refreshChurches) refreshChurches.addEventListener('click', loadAdminChurches);
    const churchDenomFilter = document.getElementById('church-denom-filter');
    if (churchDenomFilter) churchDenomFilter.addEventListener('change', loadAdminChurches);
    const refreshSuggs = document.getElementById('refresh-suggestions-btn');
    if (refreshSuggs) refreshSuggs.addEventListener('click', loadAdminSuggestions);

    // Confirm modal
    const confirmCancel = document.getElementById('confirm-cancel');
    const confirmBackdrop = document.getElementById('confirm-backdrop');
    if (confirmCancel) confirmCancel.addEventListener('click', closeConfirmModal);
    if (confirmBackdrop) confirmBackdrop.addEventListener('click', closeConfirmModal);

    // Review modal
    const reviewClose = document.getElementById('review-modal-close');
    const reviewCancel = document.getElementById('review-cancel');
    const reviewBackdrop = document.getElementById('review-backdrop');
    if (reviewClose) reviewClose.addEventListener('click', closeReviewModal);
    if (reviewCancel) reviewCancel.addEventListener('click', closeReviewModal);
    if (reviewBackdrop) reviewBackdrop.addEventListener('click', closeReviewModal);

    const reviewForm = document.getElementById('review-form');
    if (reviewForm) reviewForm.addEventListener('submit', submitReview);

    await Promise.all([loadAdminUsers(), loadAdminChurches(), loadAdminSuggestions()]);
}

async function loadAdminUsers() {
    const loading = document.getElementById('users-loading');
    const wrapper = document.getElementById('users-table-wrapper');
    const tbody = document.getElementById('users-tbody');
    if (!tbody) return;

    if (loading) loading.hidden = false;
    if (wrapper) wrapper.hidden = true;

    try {
        const users = await api('/admin/users');
        if (loading) loading.hidden = true;
        if (wrapper) wrapper.hidden = false;

        tbody.innerHTML = users.map(u => `
            <tr>
                <td>${u.id}</td>
                <td>${escapeHtml(u.name)}</td>
                <td>${escapeHtml(u.email)}</td>
                <td><span class="denomination-badge">${escapeHtml(u.denomination)}</span></td>
                <td>
                    <select data-user-id="${u.id}" class="form-select form-select--sm role-select" aria-label="Change role">
                        <option value="user" ${u.role === 'user' ? 'selected' : ''}>User</option>
                        <option value="moderator" ${u.role === 'moderator' ? 'selected' : ''}>Moderator</option>
                        <option value="church_owner" ${u.role === 'church_owner' ? 'selected' : ''}>Church Owner</option>
                        <option value="community_manager" ${u.role === 'community_manager' ? 'selected' : ''}>Community Manager</option>
                        <option value="admin" ${u.role === 'admin' ? 'selected' : ''}>Admin</option>
                    </select>
                </td>
                <td>${new Date(u.created_at).toLocaleDateString()}</td>
                <td>
                    <button class="btn btn-sm btn-primary save-role-btn" data-user-id="${u.id}">Save role</button>
                </td>
            </tr>`).join('');

        tbody.querySelectorAll('.save-role-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
                const userId = btn.dataset.userId;
                const select = tbody.querySelector(`select[data-user-id="${userId}"]`);
                if (!select) return;
                try {
                    await api('/admin/users/' + userId + '/role', {
                        method: 'PUT',
                        body: JSON.stringify({ role: select.value }),
                    });
                    showToast('Role updated', 'success');
                } catch (err) {
                    showToast(err.message, 'error');
                }
            });
        });
    } catch (err) {
        if (loading) loading.hidden = true;
        showToast('Failed to load users', 'error');
    }
}

async function loadAdminChurches() {
    const loading = document.getElementById('churches-loading');
    const wrapper = document.getElementById('churches-table-wrapper');
    const tbody = document.getElementById('churches-tbody');
    if (!tbody) return;

    if (loading) loading.hidden = false;
    if (wrapper) wrapper.hidden = true;

    try {
        const denomFilter = document.getElementById('church-denom-filter')?.value || '';
        const qs = denomFilter ? `?denomination=${encodeURIComponent(denomFilter)}` : '';
        const churches = await api('/churches' + qs);
        if (loading) loading.hidden = true;
        if (wrapper) wrapper.hidden = false;

        tbody.innerHTML = churches.map(c => `
            <tr>
                <td>${c.id}</td>
                <td><a href="/church/${c.id}">${escapeHtml(c.name)}</a></td>
                <td><span class="denomination-badge">${escapeHtml(c.denomination)}</span></td>
                <td>${escapeHtml(c.address)}</td>
                <td>${c.verified ? '<span class="status-badge status-badge--approved">Verified</span>' : '<span class="status-badge status-badge--pending">Unverified</span>'}</td>
                <td>${new Date(c.created_at).toLocaleDateString()}</td>
                <td>
                    ${!c.verified ? `<button class="btn btn-sm btn-success verify-church-btn" data-id="${c.id}">Verify</button>` : ''}
                    <button class="btn btn-sm btn-danger delete-church-btn" data-id="${c.id}">Delete</button>
                </td>
            </tr>`).join('');

        tbody.querySelectorAll('.verify-church-btn').forEach(btn => {
            btn.addEventListener('click', () => verifyChurch(btn.dataset.id));
        });
        tbody.querySelectorAll('.delete-church-btn').forEach(btn => {
            btn.addEventListener('click', () => openConfirmModal(btn.dataset.id));
        });
    } catch (err) {
        if (loading) loading.hidden = true;
        showToast('Failed to load churches', 'error');
    }
}

async function loadAdminSuggestions() {
    const loading = document.getElementById('suggestions-loading');
    const wrapper = document.getElementById('suggestions-list-wrapper');
    const list = document.getElementById('suggestions-list');
    const empty = document.getElementById('suggestions-empty');
    if (!list) return;

    if (loading) loading.hidden = false;
    if (wrapper) wrapper.hidden = true;

    try {
        const statusFilter = document.getElementById('sugg-status-filter')?.value || 'pending';
        const suggestions = await api('/suggestions?status=' + statusFilter);
        if (loading) loading.hidden = true;
        if (wrapper) wrapper.hidden = false;

        // Update badge
        const badge = document.getElementById('pending-badge');
        if (badge && statusFilter === 'pending') {
            if (suggestions.length) { badge.textContent = suggestions.length; badge.hidden = false; }
            else { badge.hidden = true; }
        }

        if (!suggestions.length) {
            list.innerHTML = '';
            if (empty) empty.hidden = false;
            return;
        }
        if (empty) empty.hidden = true;

        list.innerHTML = suggestions.map(s => `
            <li class="suggestion-review-item">
                <div class="suggestion-review-meta">
                    <span class="status-badge status-badge--${s.status}">${s.status}</span>
                    <span class="suggestion-type">${escapeHtml(s.type)}</span>
                    <span class="text-muted">${new Date(s.created_at).toLocaleDateString()}</span>
                </div>
                <p><strong>${escapeHtml(s.user?.name || 'Unknown')}</strong>${s.church ? ' - ' + escapeHtml(s.church.name) : ''}</p>
                <blockquote class="suggestion-review-content">${escapeHtml(s.content)}</blockquote>
                ${s.review_note ? '<p class="text-muted">Note: ' + escapeHtml(s.review_note) + '</p>' : ''}
                ${s.status === 'pending' ? `
                <div class="suggestion-review-actions">
                    <button class="btn btn-sm btn-success approve-btn" data-id="${s.id}">Approve</button>
                    <button class="btn btn-sm btn-danger reject-btn" data-id="${s.id}">Reject</button>
                </div>` : ''}
            </li>`).join('');

        list.querySelectorAll('.approve-btn').forEach(btn => {
            btn.addEventListener('click', () => openReviewModal(btn.dataset.id, 'approved'));
        });
        list.querySelectorAll('.reject-btn').forEach(btn => {
            btn.addEventListener('click', () => openReviewModal(btn.dataset.id, 'rejected'));
        });
    } catch (err) {
        if (loading) loading.hidden = true;
        showToast('Failed to load suggestions', 'error');
    }
}

// Suggestion filter change
document.addEventListener('change', (e) => {
    if (e.target.id === 'sugg-status-filter') loadAdminSuggestions();
});

async function verifyChurch(id) {
    try {
        await api('/admin/churches/' + id + '/verify', { method: 'PUT' });
        showToast('Church verified', 'success');
        loadAdminChurches();
    } catch (err) { showToast(err.message, 'error'); }
}

let pendingDeleteChurchId = null;

function openConfirmModal(churchId) {
    pendingDeleteChurchId = churchId;
    const modal = document.getElementById('confirm-modal');
    const msg = document.getElementById('confirm-message');
    if (msg) msg.textContent = 'Are you sure you want to delete this church? This action cannot be undone.';
    if (modal) modal.hidden = false;

    const okBtn = document.getElementById('confirm-ok');
    if (okBtn) {
        okBtn.onclick = async () => {
            try {
                await api('/admin/churches/' + pendingDeleteChurchId, { method: 'DELETE' });
                showToast('Church deleted', 'success');
                loadAdminChurches();
            } catch (err) { showToast(err.message, 'error'); }
            closeConfirmModal();
        };
    }
}

function closeConfirmModal() {
    const modal = document.getElementById('confirm-modal');
    if (modal) modal.hidden = true;
    pendingDeleteChurchId = null;
}

function openReviewModal(suggestionId, action) {
    const modal = document.getElementById('review-modal');
    const idInput = document.getElementById('review-suggestion-id');
    const actionInput = document.getElementById('review-action');
    const title = document.getElementById('review-modal-title');
    const submitBtn = document.getElementById('review-submit');

    if (idInput) idInput.value = suggestionId;
    if (actionInput) actionInput.value = action;
    if (title) title.textContent = action === 'approved' ? 'Approve Suggestion' : 'Reject Suggestion';
    if (submitBtn) {
        submitBtn.textContent = action === 'approved' ? 'Approve' : 'Reject';
        submitBtn.className = action === 'approved' ? 'btn btn-success' : 'btn btn-danger';
    }
    if (modal) modal.hidden = false;
}

function closeReviewModal() {
    const modal = document.getElementById('review-modal');
    if (modal) modal.hidden = true;
    const noteInput = document.getElementById('review-note');
    if (noteInput) noteInput.value = '';
}

async function submitReview(e) {
    e.preventDefault();
    const id = document.getElementById('review-suggestion-id')?.value;
    const action = document.getElementById('review-action')?.value;
    const note = document.getElementById('review-note')?.value || '';

    try {
        await api('/suggestions/' + id, {
            method: 'PUT',
            body: JSON.stringify({ status: action, review_note: note }),
        });
        showToast('Suggestion ' + action, 'success');
        closeReviewModal();
        loadAdminSuggestions();
    } catch (err) { showToast(err.message, 'error'); }
}

// ─── DOM Helpers ─────────────────────────────────────────────────────────────

function setText(id, text) {
    const el = document.getElementById(id);
    if (el) el.textContent = text ?? '';
}

function setVal(id, val) {
    const el = document.getElementById(id);
    if (el) el.value = val ?? '';
}

function show(id) {
    const el = document.getElementById(id);
    if (el) el.hidden = false;
}

function showFormError(id, msg) {
    const el = document.getElementById(id);
    if (el) { el.textContent = msg; el.hidden = false; }
}

function showFormSuccess(id, msg) {
    const el = document.getElementById(id);
    if (el) { el.textContent = msg; el.hidden = false; }
    setTimeout(() => { if (el) el.hidden = true; }, 3000);
}

document.addEventListener('DOMContentLoaded', init);
