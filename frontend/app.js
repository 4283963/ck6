const API_BASE = 'http://localhost:8080/api'
const REFRESH_INTERVAL = 2000
const WARNING_TEMP = 65.0

let servers = []
let selectedRack = 'all'
let refreshTimer = null

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
  
  filtered.forEach(server => {
    const isWarning = server.cpu_temp > WARNING_TEMP
    const card = document.createElement('div')
    card.className = 'server-card' + (isWarning ? ' warning' : '')
    card.onclick = () => openModal(server.id)
    
    card.innerHTML = `
      <div class="status-indicator"></div>
      <div class="server-id">${server.id}</div>
      <div class="server-temp">
        ${server.cpu_temp.toFixed(1)}
        <span class="temp-unit">°C</span>
      </div>
      <div class="server-meta">
        <div class="meta-row">
          <span>风扇</span>
          <span class="meta-value">${server.fan_speed} RPM</span>
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
          <div class="detail-value">${server.fan_speed}<span class="detail-unit"> RPM</span></div>
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

async function setAllPowerLimit(limit) {
  const promises = servers.map(s => 
    fetch(`${API_BASE}/server/${s.id}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ power_limit: limit })
    }).then(res => res.json())
  )
  
  try {
    await Promise.all(promises)
    fetchServers()
  } catch (err) {
    console.error('Failed to set all power limits:', err)
  }
}

document.getElementById('modalOverlay').addEventListener('click', (e) => {
  if (e.target.id === 'modalOverlay') {
    closeModal()
  }
})

document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    closeModal()
  }
})

function init() {
  fetchServers()
  refreshTimer = setInterval(fetchServers, REFRESH_INTERVAL)
}

init()
