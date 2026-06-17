const API_BASE = 'http://localhost:8080/api'
const REFRESH_INTERVAL = 2000
const WARNING_TEMP = 65.0
const ECO_MODE_POWER_LIMIT = 250

let servers = []
let selectedRack = 'all'
let refreshTimer = null
let selectedServers = new Set()
let lastClickedServerId = null
let isSelecting = false
let silentModeEnabled = false
let silentModeStats = { limited: 0, total: 0 }

async function fetchServers() {
  try {
    const res = await fetch(`${API_BASE}/servers`)
    const data = await res.json()
    if (data.success) {
      servers = data.data
      updateConnStatus(true)
      renderGrid()
      updateStats()
      updateRackFilter()
    } else {
      updateConnStatus(false)
    }
  } catch (err) {
    console.error('Failed to fetch servers:', err)
    updateConnStatus(false)
  }
}

async function fetchSilentModeStatus() {
  try {
    const res = await fetch(`${API_BASE}/silent-mode`)
    const data = await res.json()
    if (data.success) {
      silentModeEnabled = data.data.enabled
      silentModeStats.limited = data.data.limited_servers
      silentModeStats.total = data.data.total_servers
      updateSilentModeUI()
    }
  } catch (err) {
    console.error('Failed to fetch silent mode status:', err)
  }
}

async function toggleSilentMode() {
  const toggle = document.getElementById('silentModeToggle')
  const enabled = toggle.checked
  
  try {
    const res = await fetch(`${API_BASE}/silent-mode`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled })
    })
    const data = await res.json()
    
    if (data.success) {
      silentModeEnabled = data.data.enabled
      silentModeStats.limited = data.data.limited_servers
      silentModeStats.total = data.data.total_servers
      updateSilentModeUI()
      fetchServers()
    } else {
      toggle.checked = silentModeEnabled
      alert('操作失败: ' + (data.error || '未知错误'))
    }
  } catch (err) {
    console.error('Failed to toggle silent mode:', err)
    toggle.checked = silentModeEnabled
    alert('操作失败，请检查后端服务')
  }
}

function updateSilentModeUI() {
  const toggle = document.getElementById('silentModeToggle')
  const detail = document.getElementById('silentDetail')
  const footer = document.querySelector('.footer')
  
  if (toggle.checked !== silentModeEnabled) {
    toggle.checked = silentModeEnabled
  }
  
  if (silentModeEnabled) {
    footer.classList.add('silent-active')
    detail.textContent = `已启用 · ${silentModeStats.limited} 台静音中`
  } else {
    footer.classList.remove('silent-active')
    detail.textContent = '未启用'
  }
}

function updateConnStatus(connected) {
  const dot = document.getElementById('connStatus')
  const text = document.getElementById('connText')
  
  if (connected) {
    dot.className = 'status-dot connected'
    text.textContent = '已连接'
  } else {
    dot.className = 'status-dot error'
    text.textContent = '连接失败'
  }
}

function getRack(id) {
  return id.charAt(0)
}

function getRacks() {
  const racks = new Set()
  servers.forEach(s => racks.add(getRack(s.id)))
  return Array.from(racks).sort()
}

function updateRackFilter() {
  const container = document.getElementById('rackFilter')
  const racks = getRacks()
  
  const allBtn = document.createElement('button')
  allBtn.className = 'rack-btn' + (selectedRack === 'all' ? ' active' : '')
  allBtn.innerHTML = `全部机架 <span class="rack-count">${servers.length}</span>`
  allBtn.onclick = () => selectRack('all')
  
  container.innerHTML = ''
  container.appendChild(allBtn)
  
  racks.forEach(rack => {
    const count = servers.filter(s => getRack(s.id) === rack).length
    const btn = document.createElement('button')
    btn.className = 'rack-btn' + (selectedRack === rack ? ' active' : '')
    btn.innerHTML = `机架 ${rack} <span class="rack-count">${count}</span>`
    btn.onclick = () => selectRack(rack)
    container.appendChild(btn)
  })
}

function selectRack(rack) {
  selectedRack = rack
  updateRackFilter()
  renderGrid()
}

function renderGrid() {
  const grid = document.getElementById('serverGrid')
  
  let filtered = servers
  if (selectedRack !== 'all') {
    filtered = servers.filter(s => getRack(s.id) === selectedRack)
  }
  
  filtered.sort((a, b) => a.id.localeCompare(b.id))
  
  grid.innerHTML = ''
  
  filtered.forEach((server, index) => {
    const isWarning = server.cpu_temp > WARNING_TEMP
    const isSelected = selectedServers.has(server.id)
    const card = document.createElement('div')
    card.className = 'server-card' + (isWarning ? ' warning' : '') + (isSelected ? ' selected' : '')
    card.dataset.serverId = server.id
    card.dataset.index = index
    
    card.addEventListener('click', (e) => handleServerClick(e, server, index, filtered))
    card.addEventListener('dblclick', (e) => {
      e.stopPropagation()
      toggleServerSelection(server.id)
    })
    
    const fanSilent = server.fan_silent_limited
    const fanPercent = Math.round((server.fan_speed / 8000) * 100)
    
    card.innerHTML = `
      <div class="status-indicator"></div>
      ${fanSilent ? '<div class="silent-badge" title="静音模式限制中">🔇</div>' : ''}
      <div class="server-id">${server.id}</div>
      <div class="server-temp">
        ${server.cpu_temp.toFixed(1)}
        <span class="temp-unit">°C</span>
      </div>
      <div class="server-meta">
        <div class="meta-row">
          <span>风扇</span>
          <span class="meta-value ${fanSilent ? 'silent-fan' : ''}">
            ${fanSilent ? '🔇 ' : ''}${server.fan_speed} RPM
            <span style="opacity:0.6;font-size:10px">(${fanPercent}%)</span>
          </span>
        </div>
        <div class="meta-row">
          <span>功耗</span>
          <span class="meta-value">${server.power_usage.toFixed(0)} W</span>
        </div>
      </div>
    `
    
    grid.appendChild(card)
  })
}

function handleServerClick(e, server, index, filteredList) {
  e.stopPropagation()
  
  const isCtrlClick = e.ctrlKey || e.metaKey
  const isShiftClick = e.shiftKey
  
  if (isCtrlClick) {
    toggleServerSelection(server.id)
    lastClickedServerId = server.id
  } else if (isShiftClick && lastClickedServerId) {
    const lastIndex = filteredList.findIndex(s => s.id === lastClickedServerId)
    if (lastIndex !== -1) {
      const start = Math.min(lastIndex, index)
      const end = Math.max(lastIndex, index)
      for (let i = start; i <= end; i++) {
        selectedServers.add(filteredList[i].id)
      }
      updateSelectionUI()
    }
  } else {
    if (selectedServers.size > 0 && !selectedServers.has(server.id)) {
      clearSelection()
    }
    lastClickedServerId = server.id
    openModal(server.id)
  }
}

function toggleServerSelection(id) {
  if (selectedServers.has(id)) {
    selectedServers.delete(id)
  } else {
    selectedServers.add(id)
  }
  updateSelectionUI()
  renderGrid()
}

function clearSelection() {
  selectedServers.clear()
  lastClickedServerId = null
  updateSelectionUI()
  renderGrid()
}

function selectAllVisible() {
  let filtered = servers
  if (selectedRack !== 'all') {
    filtered = servers.filter(s => getRack(s.id) === selectedRack)
  }
  filtered.forEach(s => selectedServers.add(s.id))
  updateSelectionUI()
  renderGrid()
}

function updateSelectionUI() {
  const selectionInfo = document.getElementById('selectionInfo')
  const selectedCount = document.getElementById('selectedCount')
  const ecoBtn = document.getElementById('ecoModeBtn')
  const btn400 = document.getElementById('btn400')
  const btn500 = document.getElementById('btn500')
  const btn600 = document.getElementById('btn600')
  
  const hasSelection = selectedServers.size > 0
  
  if (hasSelection) {
    selectionInfo.style.display = 'flex'
    selectedCount.textContent = selectedServers.size
  } else {
    selectionInfo.style.display = 'none'
  }
  
  ecoBtn.disabled = !hasSelection
  btn400.disabled = !hasSelection
  btn500.disabled = !hasSelection
  btn600.disabled = !hasSelection
}

function updateStats() {
  const total = servers.length
  const warning = servers.filter(s => s.cpu_temp > WARNING_TEMP).length
  const totalPower = servers.reduce((sum, s) => sum + s.power_usage, 0)
  
  document.getElementById('totalCount').textContent = total
  document.getElementById('alertCount').textContent = warning
  document.getElementById('totalPower').textContent = totalPower.toFixed(0) + ' W'
}

function openModal(serverId) {
  const server = servers.find(s => s.id === serverId)
  if (!server) return
  
  const overlay = document.getElementById('modalOverlay')
  const title = document.getElementById('modalTitle')
  const body = document.getElementById('modalBody')
  
  const isWarning = server.cpu_temp > WARNING_TEMP
  const tempPercent = Math.min(100, (server.cpu_temp / 90) * 100)
  
  title.textContent = `服务器 ${server.id} 详情`
  
  body.innerHTML = `
    <div class="detail-section">
      <h3>实时状态</h3>
      <div class="detail-grid">
        <div class="detail-item">
          <div class="detail-label">CPU 温度</div>
          <div class="detail-value ${isWarning ? 'warning' : ''}">
            ${server.cpu_temp.toFixed(1)}<span class="detail-unit">°C</span>
          </div>
          <div class="temp-bar">
            <div class="temp-bar-fill ${isWarning ? 'warning' : ''}" style="width: ${tempPercent}%"></div>
          </div>
        </div>
        <div class="detail-item">
          <div class="detail-label">风扇转速</div>
          <div class="detail-value ${server.fan_silent_limited ? 'warning' : ''}">
            ${server.fan_silent_limited ? '🔇 ' : ''}${server.fan_speed}
            <span class="detail-unit">RPM (${Math.round(server.fan_speed / 80)}%)</span>
          </div>
          ${server.fan_silent_limited ? '<div style="font-size: 11px; color: #3fb950; margin-top: 4px;">静音模式限制中</div>' : ''}
        </div>
      </div>
    </div>
    
    <div class="detail-section">
      <h3>功耗管理</h3>
      <div class="detail-grid">
        <div class="detail-item">
          <div class="detail-label">当前功耗</div>
          <div class="detail-value">${server.power_usage.toFixed(1)}<span class="detail-unit"> W</span></div>
        </div>
        <div class="detail-item">
          <div class="detail-label">功耗上限</div>
          <div class="detail-value" style="color: #1f6feb;">${server.power_limit}<span class="detail-unit"> W</span></div>
        </div>
      </div>
      
      <div class="power-control" style="margin-top: 16px;">
        <div class="power-slider-container">
          <input type="range" class="power-slider" id="powerSlider" 
                 min="100" max="1000" step="10" value="${server.power_limit}">
          <div class="power-limit-display" id="powerLimitDisplay">${server.power_limit} W</div>
        </div>
        <div class="power-presets">
          <button class="preset-btn" onclick="setSliderValue(300)">300W</button>
          <button class="preset-btn" onclick="setSliderValue(500)">500W</button>
          <button class="preset-btn" onclick="setSliderValue(700)">700W</button>
          <button class="preset-btn" onclick="setSliderValue(1000)">1000W</button>
        </div>
        <button class="apply-btn" onclick="applyPowerLimit('${server.id}')">应用功耗限制</button>
      </div>
    </div>
    
    <div class="last-update">
      最后更新: ${new Date(server.last_updated).toLocaleTimeString('zh-CN')}
    </div>
  `
  
  const slider = document.getElementById('powerSlider')
  const display = document.getElementById('powerLimitDisplay')
  slider.addEventListener('input', () => {
    display.textContent = slider.value + ' W'
  })
  
  overlay.style.display = 'flex'
}

function setSliderValue(value) {
  const slider = document.getElementById('powerSlider')
  const display = document.getElementById('powerLimitDisplay')
  slider.value = value
  display.textContent = value + ' W'
}

async function applyPowerLimit(serverId) {
  const slider = document.getElementById('powerSlider')
  const limit = parseFloat(slider.value)
  
  try {
    const res = await fetch(`${API_BASE}/server/${serverId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ power_limit: limit })
    })
    const data = await res.json()
    if (data.success) {
      const idx = servers.findIndex(s => s.id === serverId)
      if (idx !== -1) {
        servers[idx] = data.data
      }
      renderGrid()
      updateStats()
      openModal(serverId)
    }
  } catch (err) {
    console.error('Failed to set power limit:', err)
    alert('设置功耗限制失败')
  }
}

function closeModal() {
  document.getElementById('modalOverlay').style.display = 'none'
}

async function setSelectedPowerLimit(limit) {
  if (selectedServers.size === 0) {
    alert('请先选择服务器')
    return
  }
  
  const ids = Array.from(selectedServers)
  await batchSetPowerLimit(ids, limit)
}

async function applyEcoMode() {
  if (selectedServers.size === 0) {
    alert('请先选择要切换到节能模式的服务器')
    return
  }
  
  const ids = Array.from(selectedServers)
  const confirmed = confirm(`确定要将选中的 ${ids.length} 台服务器切换到节能模式吗？\n功耗上限将设置为 ${ECO_MODE_POWER_LIMIT}W`)
  if (!confirmed) {
    return
  }
  
  await batchSetPowerLimit(ids, ECO_MODE_POWER_LIMIT)
}

async function batchSetPowerLimit(serverIds, limit) {
  try {
    const res = await fetch(`${API_BASE}/servers/batch-power-limit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        server_ids: serverIds,
        power_limit: limit
      })
    })
    const data = await res.json()
    
    if (data.success) {
      if (data.data.failed && data.data.failed.length > 0) {
        console.warn('部分服务器设置失败:', data.data.failed)
      }
      if (data.data.servers) {
        data.data.servers.forEach(updatedServer => {
          const idx = servers.findIndex(s => s.id === updatedServer.id)
          if (idx !== -1) {
            servers[idx] = updatedServer
          }
        })
      }
      renderGrid()
      updateStats()
      
      const successCount = data.data.updated ? data.data.updated.length : 0
      const failCount = data.data.failed ? data.data.failed.length : 0
      console.log(`批量设置完成: 成功 ${successCount} 台, 失败 ${failCount} 台`)
    } else {
      alert('设置失败: ' + (data.error || '未知错误'))
    }
  } catch (err) {
    console.error('批量设置功耗限制失败:', err)
    alert('批量设置失败，请检查后端服务是否正常')
  }
}

async function setAllPowerLimit(limit) {
  const confirmed = confirm(`确定要将所有 ${servers.length} 台服务器的功耗上限设置为 ${limit}W 吗？`)
  if (!confirmed) {
    return
  }
  
  const ids = servers.map(s => s.id)
  await batchSetPowerLimit(ids, limit)
}

document.getElementById('modalOverlay').addEventListener('click', (e) => {
  if (e.target.id === 'modalOverlay') {
    closeModal()
  }
})

document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    closeModal()
    if (selectedServers.size > 0) {
      clearSelection()
    }
  }
  
  if ((e.ctrlKey || e.metaKey) && e.key === 'a') {
    e.preventDefault()
    selectAllVisible()
  }
})

document.querySelector('.content').addEventListener('click', (e) => {
  if (e.target.classList.contains('content') || e.target.classList.contains('grid-container')) {
    if (!e.ctrlKey && !e.metaKey && !e.shiftKey) {
      clearSelection()
    }
  }
})

function init() {
  updateSelectionUI()
  fetchServers()
  fetchSilentModeStatus()
  refreshTimer = setInterval(() => {
    fetchServers()
    fetchSilentModeStatus()
  }, REFRESH_INTERVAL)
}

init()
