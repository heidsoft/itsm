/* ============================================================
   打击特效：粒子、飘字、震屏、闪屏
   ============================================================ */

MECHA.Effects = class Effects {
  constructor() {
    this.particles = [];
    this.texts = [];
    this.shakeTime = 0;
    this.shakeDur = 0;
    this.shakeAmp = 0;
    this.flashTime = 0;
    this.flashDur = 0;
    this.flashColor = '#ffffff';
    this.flashAlpha = 0.3;
  }

  clear() {
    this.particles.length = 0;
    this.texts.length = 0;
    this.shakeTime = 0;
    this.shakeDur = 0;
    this.flashTime = 0;
    this.flashDur = 0;
  }

  shake(amp, dur) {
    this.shakeAmp = Math.max(this.shakeAmp, amp);
    this.shakeDur = Math.max(this.shakeDur, dur);
    this.shakeTime = Math.max(this.shakeTime, dur);
  }

  flash(color, dur, alpha) {
    this.flashColor = color || '#ffffff';
    this.flashDur = dur;
    this.flashTime = dur;
    this.flashAlpha = alpha === undefined ? 0.3 : alpha;
  }

  spawn(x, y, opts) {
    const o = opts || {};
    this.particles.push({
      x: x,
      y: y,
      vx: o.vx || 0,
      vy: o.vy || 0,
      g: o.g === undefined ? 320 : o.g,
      life: o.life || 0.5,
      maxLife: o.life || 0.5,
      size: o.size || 2,
      color: o.color || '#ffffff',
      streak: !!o.streak
    });
  }

  /** 普通命中 / 暴击 */
  hit(x, y, dir, color, counter) {
    const count = counter ? 18 : 12;
    for (let i = 0; i < count; i++) {
      const a = MECHA.rand(-0.9, 0.9);
      const sp = MECHA.rand(80, counter ? 340 : 260);
      this.spawn(x, y, {
        vx: Math.cos(a) * sp * dir + dir * 60,
        vy: Math.sin(a) * sp - 40,
        g: 520,
        life: MECHA.rand(0.18, counter ? 0.5 : 0.38),
        size: MECHA.pick([1, 2, 2, 3]),
        color: MECHA.pick(['#ffffff', color, '#ffd166', '#fff3b0'])
      });
    }
    // 横向冲击线
    for (let i = 0; i < 3; i++) {
      this.spawn(x, y + MECHA.rand(-10, 10), {
        vx: dir * MECHA.rand(240, 420),
        vy: MECHA.rand(-30, 30),
        g: 0,
        life: 0.12,
        size: 2,
        color: '#ffffff',
        streak: true
      });
    }
    this.shake(counter ? 10 : 6, counter ? 0.24 : 0.16);
    this.flash(counter ? '#ffd7c2' : '#d8f7ff', 0.09, counter ? 0.22 : 0.12);
  }

  /** 格挡火花 */
  block(x, y, color) {
    for (let i = 0; i < 9; i++) {
      const a = MECHA.rand(-1.2, 1.2);
      const sp = MECHA.rand(60, 190);
      this.spawn(x, y, {
        vx: Math.cos(a) * sp,
        vy: Math.sin(a) * sp - 30,
        g: 420,
        life: MECHA.rand(0.12, 0.28),
        size: MECHA.pick([1, 2]),
        color: MECHA.pick(['#ffffff', color, '#9ff6ff'])
      });
    }
    this.shake(3, 0.1);
    this.flash('#aef4ff', 0.06, 0.1);
  }

  /** 破防 */
  guardBreak(x, y, color) {
    for (let i = 0; i < 22; i++) {
      const a = MECHA.rand(0, Math.PI * 2);
      const sp = MECHA.rand(90, 300);
      this.spawn(x, y, {
        vx: Math.cos(a) * sp,
        vy: Math.sin(a) * sp,
        g: 380,
        life: MECHA.rand(0.2, 0.55),
        size: MECHA.pick([1, 2, 3]),
        color: MECHA.pick(['#ffffff', color, '#ffd166'])
      });
    }
    this.shake(9, 0.28);
    this.flash('#ffe6a8', 0.12, 0.2);
  }

  /** 击破爆炸 */
  explode(x, y, color) {
    for (let i = 0; i < 46; i++) {
      const a = MECHA.rand(0, Math.PI * 2);
      const sp = MECHA.rand(40, 330);
      this.spawn(x, y - 20, {
        vx: Math.cos(a) * sp,
        vy: Math.sin(a) * sp - 60,
        g: 260,
        life: MECHA.rand(0.35, 1.15),
        size: MECHA.pick([1, 2, 3, 4]),
        color: MECHA.pick(['#ffffff', '#ffd166', '#ff8a3d', color, '#ff5a3d'])
      });
    }
    for (let i = 0; i < 14; i++) {
      const a = MECHA.rand(0, Math.PI * 2);
      this.spawn(x + MECHA.rand(-16, 16), y - MECHA.rand(0, 34), {
        vx: Math.cos(a) * MECHA.rand(10, 60),
        vy: MECHA.rand(-90, -20),
        g: -30,
        life: MECHA.rand(0.6, 1.4),
        size: MECHA.pick([2, 3, 4]),
        color: MECHA.pick(['#232b3d', '#39445c', '#1a2130'])
      });
    }
    this.shake(14, 0.45);
    this.flash('#ffffff', 0.16, 0.32);
  }

  /** 脚步扬尘 */
  dust(x, y, dir, color) {
    this.spawn(x, y, {
      vx: -dir * MECHA.rand(20, 70),
      vy: MECHA.rand(-40, -12),
      g: 90,
      life: MECHA.rand(0.2, 0.4),
      size: MECHA.pick([1, 2]),
      color: color || '#46536b'
    });
  }

  text(x, y, str, color, scale) {
    this.texts.push({
      x: x,
      y: y,
      vy: -42,
      life: 0.8,
      maxLife: 0.8,
      str: str,
      color: color || '#ffffff',
      scale: scale || 1
    });
  }

  update(dt) {
    for (let i = this.particles.length - 1; i >= 0; i--) {
      const p = this.particles[i];
      p.life -= dt;
      if (p.life <= 0) {
        this.particles.splice(i, 1);
        continue;
      }
      p.x += p.vx * dt;
      p.y += p.vy * dt;
      p.vy += p.g * dt;
      p.vx *= 1 - 1.6 * dt;
    }

    for (let i = this.texts.length - 1; i >= 0; i--) {
      const t = this.texts[i];
      t.life -= dt;
      t.y += t.vy * dt;
      t.vy *= 1 - 1.2 * dt;
      if (t.life <= 0) this.texts.splice(i, 1);
    }

    if (this.shakeTime > 0) this.shakeTime -= dt;
    if (this.shakeTime <= 0) {
      this.shakeTime = 0;
      this.shakeDur = 0;
      this.shakeAmp = 0;
    }
    if (this.flashTime > 0) this.flashTime -= dt;
  }

  getShakeOffset() {
    if (this.shakeTime <= 0 || this.shakeDur <= 0) return { x: 0, y: 0 };
    const k = this.shakeTime / this.shakeDur;
    const amp = this.shakeAmp * k;
    return {
      x: Math.round(MECHA.rand(-amp, amp)),
      y: Math.round(MECHA.rand(-amp * 0.6, amp * 0.6))
    };
  }

  getFlashAlpha() {
    if (this.flashTime <= 0 || this.flashDur <= 0) return 0;
    return this.flashAlpha * (this.flashTime / this.flashDur);
  }

  draw(ctx) {
    for (let i = 0; i < this.particles.length; i++) {
      const p = this.particles[i];
      const alpha = Math.min(1, (p.life / p.maxLife) * 2.2);
      ctx.globalAlpha = alpha;
      ctx.fillStyle = p.color;
      if (p.streak) {
        ctx.fillRect(Math.round(p.x), Math.round(p.y), Math.round(p.size * 4), p.size);
      } else {
        ctx.fillRect(Math.round(p.x), Math.round(p.y), p.size, p.size);
      }
    }
    ctx.globalAlpha = 1;

    for (let i = 0; i < this.texts.length; i++) {
      const t = this.texts[i];
      const alpha = Math.min(1, t.life * 3);
      ctx.globalAlpha = alpha;
      MECHA.drawText(ctx, t.str, Math.round(t.x), Math.round(t.y), t.scale, t.color, {
        align: 'center',
        shadow: '#050810'
      });
    }
    ctx.globalAlpha = 1;
  }
};
