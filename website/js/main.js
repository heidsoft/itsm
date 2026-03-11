// ITSM 官网主脚本

// 平滑滚动
document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function (e) {
        e.preventDefault();
        const target = document.querySelector(this.getAttribute('href'));
        if (target) {
            target.scrollIntoView({
                behavior: 'smooth',
                block: 'start'
            });
        }
    });
});

// 导航栏滚动效果
window.addEventListener('scroll', function() {
    const navbar = document.querySelector('.navbar');
    if (window.scrollY > 50) {
        navbar.style.boxShadow = '0 4px 8px rgba(0,0,0,0.15)';
    } else {
        navbar.style.boxShadow = '0 2px 4px rgba(0,0,0,0.1)';
    }
});

// 统计数字动画
function animateStats() {
    const stats = document.querySelectorAll('.stat-number');
    stats.forEach(stat => {
        const target = stat.textContent;
        const number = parseInt(target.replace(/[^0-9]/g, ''));
        const suffix = target.replace(/[0-9]/g, '');
        
        let current = 0;
        const increment = number / 50;
        const timer = setInterval(() => {
            current += increment;
            if (current >= number) {
                stat.textContent = number + suffix;
                clearInterval(timer);
            } else {
                stat.textContent = Math.floor(current) + suffix;
            }
        }, 30);
    });
}

// 当统计区域可见时触发动画
const statsSection = document.querySelector('.hero-stats');
if (statsSection) {
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                animateStats();
                observer.unobserve(entry.target);
            }
        });
    }, { threshold: 0.5 });
    
    observer.observe(statsSection);
}

// 功能卡片动画
const featureCards = document.querySelectorAll('.feature-card');
const cardObserver = new IntersectionObserver((entries) => {
    entries.forEach((entry, index) => {
        if (entry.isIntersecting) {
            setTimeout(() => {
                entry.target.style.opacity = '1';
                entry.target.style.transform = 'translateY(0)';
            }, index * 100);
            cardObserver.unobserve(entry.target);
        }
    });
}, { threshold: 0.1 });

featureCards.forEach(card => {
    card.style.opacity = '0';
    card.style.transform = 'translateY(20px)';
    card.style.transition = 'all 0.5s ease-out';
    cardObserver.observe(card);
});

// 表单提交处理
const trialForm = document.getElementById('trialForm');
if (trialForm) {
    trialForm.addEventListener('submit', function(e) {
        e.preventDefault();
        
        // 获取表单数据
        const formData = new FormData(this);
        const data = Object.fromEntries(formData);
        
        // 显示成功消息
        alert('感谢您的申请！我们的销售顾问会在 1 个工作日内与您联系。');
        
        // 重置表单
        this.reset();
        
        // 这里可以添加实际的 API 调用
        console.log('试用申请:', data);
    });
}

const contactForm = document.getElementById('contactForm');
if (contactForm) {
    contactForm.addEventListener('submit', function(e) {
        e.preventDefault();
        
        // 获取表单数据
        const formData = new FormData(this);
        const data = Object.fromEntries(formData);
        
        // 显示成功消息
        alert('感谢您的留言！我们会尽快回复您。');
        
        // 重置表单
        this.reset();
        
        // 这里可以添加实际的 API 调用
        console.log('联系表单:', data);
    });
}

// 控制台输出
console.log('%c ITSM 官网 ', 'background: #1890FF; color: #fff; font-size: 20px; padding: 10px;');
console.log('%c 企业级 IT 服务管理平台 ', 'color: #666; font-size: 14px;');
console.log('%c 免费试用：#trial ', 'color: #1890FF; font-size: 14px;');

// 试用表单提交处理
const trialForm = document.getElementById('trialForm');
const formSuccess = document.getElementById('formSuccess');

if (trialForm) {
    trialForm.addEventListener('submit', function(e) {
        // 如果使用 Formspree，让表单正常提交
        // Formspree 会自动处理邮件发送和重定向
        
        // 显示加载状态
        const submitBtn = trialForm.querySelector('button[type="submit"]');
        const originalText = submitBtn.innerHTML;
        submitBtn.innerHTML = '⏳ 提交中...';
        submitBtn.disabled = true;
        
        // Formspree 会处理提交，这里可以添加自定义逻辑
        console.log('表单提交中...');
    });
    
    // 检查 URL 参数（Formspree 提交后会重定向）
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get('status') === 'success') {
        trialForm.style.display = 'none';
        formSuccess.style.display = 'block';
        // 清除 URL 参数
        window.history.replaceState({}, document.title, window.location.pathname);
    }
}
