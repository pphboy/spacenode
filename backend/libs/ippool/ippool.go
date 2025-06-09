package ippool

import (
	"errors"
	"math/rand"
	"net"
	"sync"
	"time"
)

// IPPool 结构体表示一个 IP 地址池
type IPPool struct {
	mu          sync.Mutex
	startIP     net.IP
	mask        net.IPMask
	network     *net.IPNet
	allocated   map[string]time.Time // 记录已分配的IP及其过期时间
	available   []net.IP             // 可用的IP列表
	initialized bool
}

// NewIPPool 创建一个新的IP池
func NewIPPool(startIP string, mask string) (*IPPool, error) {
	ip := net.ParseIP(startIP)
	if ip == nil {
		return nil, errors.New("invalid start IP address")
	}

	parsedMask := net.ParseIP(mask)
	if parsedMask == nil {
		return nil, errors.New("invalid mask")
	}

	m := net.IPMask(parsedMask.To4())
	if m == nil {
		return nil, errors.New("invalid mask format")
	}

	network := &net.IPNet{
		IP:   ip.Mask(m),
		Mask: m,
	}

	pool := &IPPool{
		startIP:     ip,
		mask:        m,
		network:     network,
		allocated:   make(map[string]time.Time),
		available:   make([]net.IP, 0),
		initialized: false,
	}

	// 初始化可用IP列表
	if err := pool.initialize(); err != nil {
		return nil, err
	}

	return pool, nil
}

// initialize 初始化IP池，生成所有可用的IP地址
func (p *IPPool) initialize() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return nil
	}

	// 计算网络中的IP数量
	ones, bits := p.mask.Size()
	if ones == 0 || bits == 0 {
		return errors.New("invalid mask")
	}

	totalIPs := 1 << uint(bits-ones)
	if totalIPs <= 2 { // 网络地址和广播地址
		return errors.New("network too small")
	}

	// 生成所有可用的IP地址（排除网络地址和广播地址）
	ip := p.startIP.Mask(p.mask)
	for i := 1; i < totalIPs-1; i++ {
		nextIP := make(net.IP, len(ip))
		copy(nextIP, ip)

		// 增加IP
		for j := len(nextIP) - 1; j >= 0; j-- {
			nextIP[j]++
			if nextIP[j] > 0 {
				break
			}
		}

		// 检查是否超出网络范围
		if !p.network.Contains(nextIP) {
			break
		}

		p.available = append(p.available, nextIP)
		ip = nextIP
	}

	p.initialized = true
	return nil
}

// Random 随机分配一个IP地址，并指定存活时间
func (p *IPPool) Random(ttl time.Duration) (net.IP, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return nil, errors.New("IP pool not initialized")
	}

	// 清理过期的IP
	p.cleanExpired()

	if len(p.available) == 0 {
		return nil, errors.New("no available IP addresses")
	}

	// 随机选择一个IP
	idx := rand.Intn(len(p.available))
	ip := p.available[idx]

	// 从可用列表中移除
	p.available = append(p.available[:idx], p.available[idx+1:]...)

	// 记录到已分配列表
	p.allocated[ip.String()] = time.Now().Add(ttl)

	return ip, nil
}

// RequestIP 申请指定的IP地址，并指定存活时间
func (p *IPPool) RequestIP(ipStr string, ttl time.Duration) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return false, errors.New("IP pool not initialized")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, errors.New("invalid IP address")
	}

	// 检查IP是否在范围内
	if !p.network.Contains(ip) {
		return false, errors.New("IP address not in pool range")
	}

	// 检查IP是否已被分配
	if expiry, exists := p.allocated[ip.String()]; exists {
		if time.Now().Before(expiry) {
			return false, nil // IP已被分配且未过期
		}
		// IP已过期，可以重新分配
	}

	// 检查IP是否在可用列表中
	found := false
	for i, availableIP := range p.available {
		if availableIP.Equal(ip) {
			// 从可用列表中移除
			p.available = append(p.available[:i], p.available[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return false, nil
	}

	// 记录到已分配列表
	p.allocated[ip.String()] = time.Now().Add(ttl)

	return true, nil
}

// CleanIP 清理指定的IP地址，使其重新可用
func (p *IPPool) CleanIP(ipStr string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return errors.New("IP pool not initialized")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return errors.New("invalid IP address")
	}

	// 检查IP是否在范围内
	if !p.network.Contains(ip) {
		return errors.New("IP address not in pool range")
	}

	// 检查IP是否已被分配
	if _, exists := p.allocated[ip.String()]; exists {
		delete(p.allocated, ip.String())
		// 添加到可用列表
		p.available = append(p.available, ip)
		return nil
	}

	// 检查IP是否已经在可用列表中
	for _, availableIP := range p.available {
		if availableIP.Equal(ip) {
			return nil // 已经在可用列表中
		}
	}

	// 如果既不在已分配列表也不在可用列表，添加到可用列表
	p.available = append(p.available, ip)
	return nil
}

// CheckIP 检查IP状态，返回是否被占用和剩余存活时间
func (p *IPPool) CheckIP(ipStr string) (bool, time.Duration, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return false, 0, errors.New("IP pool not initialized")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, 0, errors.New("invalid IP address")
	}

	// 检查IP是否在范围内
	if !p.network.Contains(ip) {
		return false, 0, errors.New("IP address not in pool range")
	}

	// 检查IP是否已被分配
	if expiry, exists := p.allocated[ip.String()]; exists {
		if time.Now().Before(expiry) {
			return true, time.Until(expiry), nil
		}
		// IP已过期，视为可用
		return false, 0, nil
	}

	// 检查IP是否在可用列表中
	for _, availableIP := range p.available {
		if availableIP.Equal(ip) {
			return false, 0, nil
		}
	}

	// 如果既不在已分配列表也不在可用列表，视为可用
	return false, 0, nil
}

// cleanExpired 清理过期的IP地址
func (p *IPPool) cleanExpired() {
	now := time.Now()
	for ipStr, expiry := range p.allocated {
		if now.After(expiry) {
			delete(p.allocated, ipStr)
			ip := net.ParseIP(ipStr)
			if ip != nil {
				p.available = append(p.available, ip)
			}
		}
	}
}
