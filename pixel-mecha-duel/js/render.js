/* ============================================================
   渲染层：像素字体、场景、机甲绘制、HUD、回合横幅
   ============================================================ */

(function () {
  const CFG = MECHA.CFG;
  const OUTLINE = '#070c15';

  function px(ctx, x, y, w, h, color) {
    ctx.fillStyle = color;
    ctx.fillRect(Math.round(x), Math.round(y), Math.max(1, Math.round(w)), Math.max(1, Math.round(h)));
  }

  /* ------------------------------------------------------------
     5x7 像素字体
     ------------------------------------------------------------ */
  MECHA.drawText = function (ctx, text, x, y, scale, color, opts) {
    const o = opts || {};
    const s = scale || 1;
    const str = String(text).toUpperCase();
    const glyphW = 5 * s;
    const gap = s;
    const totalW = str.length * (glyphW + gap) - gap;
    let startX = x;
    if (o.align === 'center') startX = x - Math.floor(totalW / 2);
    else if (o.align === 'right') startX = x - totalW;

    const pass = function (ox, oy, col) {
      ctx.fillStyle = col;
      for (let i = 0; i < str.length; i++) {
        const glyph = MECHA.FONT[str[i]] || MECHA.FONT[' '];
        const gx = startX + i * (glyphW + gap) + ox;
        for (let row = 0; row < 7; row++) {
          const bits = glyph[row];
          for (let col2 = 0; col2 < 5; col2++) {
            if (bits[col2] === '1') {
              ctx.fillRect(Math.round(gx + col2 * s), Math.round(y + oy + row * s), s, s);
            }
          }
        }
      }
    };

    if (o.shadow) {
      const off = o.shadowOffset === undefined ? s : o.shadowOffset;
      pass(off, off, o.shadow);
    }
    pass(0, 0, color);
    return totalW;
  };

  /* ------------------------------------------------------------
     像素月亮
     ------------------------------------------------------------ */
  function drawMoon(g, cx, cy, r) {
    g.fillStyle = 'rgba(244,236,201,0.12)';
    g.beginPath();
    g.arc(cx, cy, r + 10, 0, Math.PI * 2);
    g.fill();

    g.fillStyle = '#f2e9c4';
    for (let dy = -r; dy < r; dy += 3) {
      const half = Math.sqrt(Math.max(0, r * r - dy * dy));
      const w = Math.max(2, Math.round(half / 1) * 2);
      g.fillRect(Math.round(cx - w / 2), Math.round(cy + dy), w, 3);
    }
    g.fillStyle = '#cfc6a4';
    g.fillRect(cx - 12, cy - 8, 5, 4);
    g.fillRect(cx + 3, cy + 1, 8, 5);
    g.fillRect(cx - 4, cy + 10, 6, 3);
    g.fillRect(cx + 9, cy - 12, 3, 3);
  }

  /* ------------------------------------------------------------
     城市剪影
     ------------------------------------------------------------ */
  function drawSkyline(g, rnd, baseY, maxH, body, windowColor, chance) {
    let x = -12;
    while (x < 652) {
      const w = 16 + Math.floor(rnd() * 34);
      const h = 24 + Math.floor(rnd() * maxH);
      const top = baseY - h;
      g.fillStyle = body;
      g.fillRect(x, top, w, h + 20);

      if (rnd() < 0.3) {
        g.fillRect(x + Math.floor(w * 0.5), top - 8, 1, 8);
        g.fillStyle = '#ff5566';
        g.fillRect(x + Math.floor(w * 0.5), top - 9, 1, 1);
      }

      for (let wy = top + 6; wy < baseY - 4; wy += 8) {
        for (let wx = x + 4; wx < x + w - 3; wx += 7) {
          if (rnd() < chance) {
            g.globalAlpha = 0.18 + rnd() * 0.5;
            g.fillStyle = windowColor;
            g.fillRect(wx, wy, 2, 2);
          }
        }
      }
      g.globalAlpha = 1;
      x += w + 2 + Math.floor(rnd() * 7);
    }
  }

  /* ------------------------------------------------------------
     搭建静态背景（只执行一次）
     ------------------------------------------------------------ */
  function buildBackground() {
    const bg = document.createElement('canvas');
    bg.width = CFG.W;
    bg.height = CFG.H;
    const g = bg.getContext('2d');
    g.imageSmoothingEnabled = false;
    const rnd = MECHA.seeded(19890417);

    // 夜空分层
    const bands = [
      [0, 46, '#04060f'],
      [46, 40, '#060a1a'],
      [86, 40, '#080f26'],
      [126, 44, '#0b1533'],
      [170, 46, '#0e1b3f'],
      [216, 88, '#0a1229']
    ];
    bands.forEach(function (b) {
      g.fillStyle = b[2];
      g.fillRect(0, b[0], CFG.W, b[1]);
    });

    // 星空
    for (let i = 0; i < 140; i++) {
      const sx = Math.floor(rnd() * CFG.W);
      const sy = Math.floor(rnd() * 215);
      const size = rnd() < 0.1 ? 2 : 1;
      g.globalAlpha = 0.2 + rnd() * 0.7;
      g.fillStyle = rnd() < 0.12 ? '#ffd9f4' : rnd() < 0.3 ? '#9fe8ff' : '#cfe0ff';
      g.fillRect(sx, sy, size, size);
    }
    g.globalAlpha = 1;

    drawMoon(g, 512, 60, 23);

    // 远景 / 近景城市
    drawSkyline(g, rnd, 304, 128, '#081022', '#1b2f5c', 0.3);
    drawSkyline(g, rnd, 304, 82, '#050a16', '#4a63a8', 0.42);

    // 城市霓虹灯牌
    for (let i = 0; i < 14; i++) {
      const nx = Math.floor(rnd() * CFG.W);
      const ny = 200 + Math.floor(rnd() * 90);
      const nh = 6 + Math.floor(rnd() * 16);
      g.globalAlpha = 0.5;
      g.fillStyle = rnd() < 0.5 ? '#ff4fc8' : '#4be4ff';
      g.fillRect(nx, ny, rnd() < 0.3 ? 2 : 1, nh);
    }
    g.globalAlpha = 1;

    // 地平线雾气
    const haze = g.createLinearGradient(0, 246, 0, 304);
    haze.addColorStop(0, 'rgba(10,20,45,0)');
    haze.addColorStop(1, 'rgba(14,28,58,0.55)');
    g.fillStyle = haze;
    g.fillRect(0, 246, CFG.W, 58);

    // 顶棚钢架
    g.fillStyle = '#060a14';
    g.fillRect(0, 0, CFG.W, 18);
    g.fillStyle = '#101a2c';
    g.fillRect(0, 0, CFG.W, 2);
    g.fillStyle = '#1b2c45';
    for (let tx = 8; tx < CFG.W; tx += 38) {
      g.fillRect(tx, 2, 2, 15);
    }
    g.strokeStyle = '#0d1727';
    g.lineWidth = 1;
    for (let tx = 8; tx < CFG.W - 40; tx += 38) {
      g.beginPath();
      g.moveTo(tx + 2, 3);
      g.lineTo(tx + 38, 16);
      g.stroke();
      g.beginPath();
      g.moveTo(tx + 2, 16);
      g.lineTo(tx + 38, 3);
      g.stroke();
    }

    // 吊索 + 霓虹招牌
    g.fillStyle = '#26354f';
    g.fillRect(216, 18, 2, 48);
    g.fillRect(424, 18, 2, 48);

    g.fillStyle = '#050912';
    g.fillRect(190, 66, 260, 46);
    g.fillStyle = '#0b1424';
    g.fillRect(192, 68, 256, 42);

    // 招牌霓虹边
    g.globalAlpha = 0.85;
    g.fillStyle = '#ff4fc8';
    g.fillRect(192, 68, 256, 1);
    g.fillRect(192, 109, 256, 1);
    g.fillRect(192, 68, 1, 42);
    g.fillRect(447, 68, 1, 42);
    g.globalAlpha = 1;

    // 招牌四角螺栓
    g.fillStyle = '#7de9ff';
    [[194, 70], [445, 70], [194, 106], [445, 106]].forEach(function (p) {
      g.fillRect(p[0], p[1], 2, 2);
    });

    MECHA.drawText(g, 'MECHA DUEL', 320, 74, 3, '#ffe6f9', {
      align: 'center', shadow: '#ff2fbf', shadowOffset: 2
    });
    MECHA.drawText(g, '1P  VS  2P', 320, 99, 1, '#8deaff', {
      align: 'center', shadow: '#04121c'
    });

    // 两侧垂线与壁灯
    g.fillStyle = '#1b2c45';
    g.fillRect(6, 18, 2, 286);
    g.fillRect(632, 18, 2, 286);
    for (let ly = 60; ly < 280; ly += 54) {
      g.fillStyle = '#4be4ff';
      g.globalAlpha = 0.5;
      g.fillRect(4, ly, 6, 2);
      g.globalAlpha = 0.5;
      g.fillStyle = '#ff4fc8';
      g.fillRect(630, ly + 20, 6, 2);
      g.globalAlpha = 1;
    }

    // 地面
    g.fillStyle = '#060a13';
    g.fillRect(0, 304, CFG.W, 56);
    g.fillStyle = '#080f1c';
    g.fillRect(0, 304, CFG.W, 18);
    g.fillStyle = '#050810';
    g.fillRect(0, 322, CFG.W, 38);

    // 台沿高光
    g.fillStyle = '#22344f';
    g.fillRect(0, 301, CFG.W, 3);
    g.globalAlpha = 0.55;
    g.fillStyle = '#4be4ff';
    g.fillRect(0, 301, CFG.W, 1);
    g.globalAlpha = 1;

    // 透视网格
    g.strokeStyle = 'rgba(34,58,92,0.6)';
    g.lineWidth = 1;
    const vpX = 320;
    const vpY = 238;
    for (let i = -12; i <= 12; i++) {
      g.beginPath();
      g.moveTo(vpX + i * 20, vpY);
      g.lineTo(vpX + i * 96, 360);
      g.stroke();
    }
    [312, 318, 326, 336, 348, 360].forEach(function (yy) {
      g.beginPath();
      g.moveTo(0, yy);
      g.lineTo(CFG.W, yy);
      g.stroke();
    });

    // 地面前沿警示条
    g.globalAlpha = 0.32;
    for (let sx = 0; sx < CFG.W; sx += 18) {
      g.fillStyle = '#ffd166';
      g.fillRect(sx, 349, 9, 3);
    }
    g.globalAlpha = 1;

    // 霓虹地面反光
    g.globalAlpha = 0.1;
    g.fillStyle = '#ff4fc8';
    g.fillRect(250, 305, 140, 32);
    g.globalAlpha = 0.07;
    g.fillStyle = '#4be4ff';
    g.fillRect(90, 305, 80, 20);
    g.fillRect(470, 305, 80, 20);
    g.globalAlpha = 1;

    return bg;
  }

  /* ------------------------------------------------------------
     机甲绘制
     ------------------------------------------------------------ */
  function drawWreck(ctx, f, pal, time) {
    ctx.save();
    ctx.translate(Math.round(f.x), CFG.GROUND);
    ctx.scale(f.facing, 1);

    px(ctx, -27, -6, 18, 5, pal.metalDark);
    px(ctx, 9, -5, 17, 5, pal.metal);
    px(ctx, -17, -23, 30, 18, OUTLINE);
    px(ctx, -15, -21, 26, 15, pal.shellDark);
    px(ctx, -8, -19, 13, 8, pal.metal);
    px(ctx, -4, -17, 6, 5, '#0a0f18');
    px(ctx, -2, -16, 2, 2, pal.accent);
    px(ctx, 7, -26, 15, 12, OUTLINE);
    px(ctx, 8, -25, 13, 10, pal.metal);
    px(ctx, 14, -21, 6, 2, '#15202e');
    px(ctx, 8, -27, 13, 2, pal.shellDark);
    px(ctx, -22, -13, 11, 5, pal.metalDark);
    px(ctx, -24, -9, 5, 3, OUTLINE);

    // 故障电火花
    if (Math.floor(time * 9) % 5 === 0) {
      px(ctx, -3, -18, 2, 2, '#ffd166');
      px(ctx, 3, -24, 1, 1, '#ffffff');
    } else if (Math.floor(time * 7) % 7 === 0) {
      px(ctx, 10, -24, 2, 1, '#8deaff');
    }
    ctx.restore();
  }

  function drawPalette(f) {
    const c = f.colors;
    if (f.hitFlash > 0) {
      return {
        shell: '#ffffff',
        shellDark: '#cfe9ff',
        plate: '#ffffff',
        metal: '#dcefff',
        metalDark: '#9fc4e0',
        accent: '#ffffff',
        eye: '#ffffff',
        flame: '#ffffff'
      };
    }
    return {
      shell: c.shell,
      shellDark: c.shellDark,
      plate: c.plate,
      metal: c.metal,
      metalDark: c.metalDark,
      accent: c.accent,
      eye: c.eye,
      flame: c.flame
    };
  }

  MECHA.drawMecha = function (ctx, f, time) {
    const pal = drawPalette(f);

    // 地面阴影
    ctx.globalAlpha = 0.34;
    ctx.fillStyle = '#000000';
    ctx.fillRect(Math.round(f.x - 18), CFG.GROUND - 3, 36, 4);
    ctx.fillRect(Math.round(f.x - 10), CFG.GROUND - 5, 20, 2);
    ctx.globalAlpha = 1;

    if (f.state === 'ko') {
      drawWreck(ctx, f, pal, time);
      return;
    }

    const walking = f.state === 'walk';
    const guarding = f.state === 'guard';
    const attacking = f.state === 'attack';
    const phase = f.getAttackPhase();

    let bob = 0;
    if (walking) bob = -Math.abs(Math.sin(f.walkPhase) * 1.5);
    else if (f.state === 'idle') bob = Math.round(Math.sin(f.animT * 3.4));
    else if (guarding) bob = 2;

    let lean = 0;
    if (attacking) lean = phase === 'startup' ? -2 : 2;
    if (f.state === 'hit') lean = -2;
    if (f.state === 'guardBreak') lean = -3;

    const swingRaw = walking ? Math.sin(f.walkPhase) * 3.4 : 0;
    const swing = Math.round(swingRaw);
    const liftF = walking ? Math.round(Math.max(0, Math.sin(f.walkPhase)) * 3) : 0;
    const liftB = walking ? Math.round(Math.max(0, -Math.sin(f.walkPhase)) * 3) : 0;

    ctx.save();
    ctx.translate(Math.round(f.x + lean), Math.round(CFG.GROUND + bob));
    ctx.scale(f.facing, 1);

    // 背包 + 推进器火焰
    px(ctx, -19, -50, 6, 16, OUTLINE);
    px(ctx, -18, -48, 4, 12, pal.metalDark);
    px(ctx, -18, -44, 4, 2, pal.accent);

    const flick = Math.floor(time * 22) % 3;
    px(ctx, -23 - flick, -37, 3, 3, pal.flame);
    px(ctx, -26 - flick, -34, 3, 2, pal.accent);
    if (flick !== 2) px(ctx, -24, -31, 2, 2, pal.flame);

    // 后侧手臂 + 肩甲
    px(ctx, -17, -47, 5, 11, OUTLINE);
    px(ctx, -16, -46, 3, 9, pal.metalDark);
    px(ctx, -19, -51, 9, 11, OUTLINE);
    px(ctx, -18, -50, 7, 9, pal.shellDark);

    // 后腿
    px(ctx, -11 + Math.round(swing * 0.5), -25, 7, 13, OUTLINE);
    px(ctx, -10 + Math.round(swing * 0.5), -24, 5, 12, pal.metalDark);
    px(ctx, -12 + swing, -15, 6, 12, OUTLINE);
    px(ctx, -11 + swing, -14, 4, 11, pal.metal);
    px(ctx, -14 + swing, -4 - liftB, 10, 4, OUTLINE);
    px(ctx, -13 + swing, -4 - liftB, 8, 3, pal.metalDark);

    // 前腿
    px(ctx, 4 + Math.round(swing * 0.5), -25, 7, 13, OUTLINE);
    px(ctx, 5 + Math.round(swing * 0.5), -24, 5, 12, pal.metal);
    px(ctx, 5 + swing, -15, 6, 12, OUTLINE);
    px(ctx, 6 + swing, -14, 4, 11, pal.metalDark);
    px(ctx, 3 + swing, -4 - liftF, 11, 4, OUTLINE);
    px(ctx, 4 + swing, -4 - liftF, 9, 3, pal.metal);

    // 骨盆
    px(ctx, -9, -32, 18, 8, OUTLINE);
    px(ctx, -8, -31, 16, 6, pal.metal);
    px(ctx, -3, -31, 6, 3, pal.shellDark);

    // 躯干
    px(ctx, -12, -51, 24, 22, OUTLINE);
    px(ctx, -11, -50, 22, 20, pal.metal);
    px(ctx, -9, -48, 18, 13, pal.shell);
    px(ctx, -9, -48, 18, 2, pal.plate);
    px(ctx, -8, -37, 16, 4, pal.shellDark);
    px(ctx, -2, -45, 6, 6, OUTLINE);
    px(ctx, -1, -44, 4, 4, pal.accent);
    px(ctx, 0, -43, 2, 2, pal.eye);

    // 前肩甲
    px(ctx, 9, -51, 11, 12, OUTLINE);
    px(ctx, 10, -50, 9, 10, pal.shell);
    px(ctx, 10, -50, 9, 2, pal.plate);
    px(ctx, 13, -46, 3, 3, pal.shellDark);

    // 头部
    px(ctx, -3, -55, 6, 5, OUTLINE);
    px(ctx, -8, -66, 16, 12, OUTLINE);
    px(ctx, -7, -65, 14, 10, pal.metal);
    px(ctx, -7, -65, 14, 3, pal.shell);
    px(ctx, -2, -61, 9, 3, pal.eye);
    px(ctx, 5, -61, 2, 2, pal.plate);
    px(ctx, -7, -57, 10, 2, pal.shellDark);
    px(ctx, 4, -70, 2, 6, pal.shell);
    px(ctx, 4, -72, 2, 3, pal.accent);

    // 前臂 / 攻击动作
    if (attacking) {
      const reach = phase === 'startup' ? 3 : phase === 'active' ? 16 : 8;
      px(ctx, 10, -48, 11, 8, OUTLINE);
      px(ctx, 11, -47, 9, 6, pal.metal);
      px(ctx, 19, -46, 8 + reach, 7, OUTLINE);
      px(ctx, 20, -45, 7 + reach, 5, pal.shell);
      const fistX = 27 + reach;
      px(ctx, fistX, -49, 10, 10, OUTLINE);
      px(ctx, fistX + 1, -48, 8, 8, pal.plate);
      px(ctx, fistX + 2, -47, 6, 2, pal.metal);

      if (phase === 'active') {
        ctx.globalAlpha = 0.55;
        ctx.fillStyle = '#ffffff';
        ctx.fillRect(17, -44, 12, 1);
        ctx.fillRect(13, -39, 9, 1);
        ctx.globalAlpha = 1;
      }
    } else if (guarding) {
      px(ctx, 8, -49, 8, 14, OUTLINE);
      px(ctx, 9, -48, 6, 12, pal.metal);
      px(ctx, 7, -38, 11, 8, OUTLINE);
      px(ctx, 8, -37, 9, 6, pal.shell);
    } else {
      px(ctx, 10, -48, 6, 11, OUTLINE);
      px(ctx, 11, -47, 4, 9, pal.metal);
      px(ctx, 10, -38, 7, 5, OUTLINE);
      px(ctx, 11, -38, 5, 3, pal.shellDark);
      px(ctx, 8, -33, 9, 8, OUTLINE);
      px(ctx, 9, -32, 7, 6, pal.metal);
      px(ctx, 9, -32, 7, 2, pal.shell);
    }

    // 能量护盾
    if (guarding && f.guard > 0) {
      const sx = 15;
      const sy = -64;
      const sw = 8;
      const sh = 60;
      ctx.globalAlpha = 0.16 + (f.guard / 100) * 0.12;
      ctx.fillStyle = pal.shell;
      ctx.fillRect(sx, sy, sw, sh);
      ctx.globalAlpha = 0.45;
      ctx.fillStyle = pal.plate;
      ctx.fillRect(sx + sw - 2, sy, 2, sh);
      const scanY = sy + ((time * 58) % sh);
      ctx.globalAlpha = 0.35;
      ctx.fillRect(sx, Math.round(scanY), sw, 2);
      ctx.globalAlpha = 1;
    }

    ctx.restore();
  };

  /* ------------------------------------------------------------
     Renderer
     ------------------------------------------------------------ */
  MECHA.Renderer = class Renderer {
    constructor(canvas) {
      this.canvas = canvas;
      this.ctx = canvas.getContext('2d');
      this.ctx.imageSmoothingEnabled = false;
      this.bg = buildBackground();
      this.motes = [];
      const rnd = MECHA.seeded(77123);
      for (let i = 0; i < 42; i++) {
        this.motes.push({
          x: rnd() * CFG.W,
          y: rnd() * CFG.H,
          speed: 3 + rnd() * 9,
          phase: rnd() * Math.PI * 2,
          size: rnd() < 0.25 ? 2 : 1,
          alpha: 0.08 + rnd() * 0.28
        });
      }
    }

    drawAmbient(ctx, time, state) {
      // 招牌光晕
      const pulse = 0.85 + Math.sin(time * 2.1) * 0.15;
      const halo = ctx.createRadialGradient(320, 90, 4, 320, 90, 178);
      halo.addColorStop(0, 'rgba(255,79,200,' + (0.17 * pulse).toFixed(3) + ')');
      halo.addColorStop(0.45, 'rgba(75,228,255,' + (0.055 * pulse).toFixed(3) + ')');
      halo.addColorStop(1, 'rgba(0,0,0,0)');
      ctx.fillStyle = halo;
      ctx.fillRect(130, 0, 380, 230);

      // 顶部射灯
      const beam = 0.055 + Math.sin(time * 1.6) * 0.012;
      ctx.globalAlpha = beam;
      ctx.fillStyle = '#9fe8ff';
      ctx.beginPath();
      ctx.moveTo(64, 18);
      ctx.lineTo(206, 304);
      ctx.lineTo(292, 304);
      ctx.closePath();
      ctx.fill();
      ctx.beginPath();
      ctx.moveTo(576, 18);
      ctx.lineTo(348, 304);
      ctx.lineTo(434, 304);
      ctx.closePath();
      ctx.fill();
      ctx.globalAlpha = 1;

      // 漂浮尘埃
      for (let i = 0; i < this.motes.length; i++) {
        const m = this.motes[i];
        const yy = ((m.y - time * m.speed) % CFG.H + CFG.H) % CFG.H;
        const xx = m.x + Math.sin(time * 0.7 + m.phase) * 9;
        ctx.globalAlpha = m.alpha * (0.6 + 0.4 * Math.sin(time * 2 + m.phase));
        ctx.fillStyle = '#9fdcff';
        ctx.fillRect(Math.round(xx), Math.round(yy), m.size, m.size);
      }
      ctx.globalAlpha = 1;

      // 边灯闪烁
      if (Math.floor(time * 3) % 7 !== 0) {
        ctx.globalAlpha = 0.32;
        ctx.fillStyle = '#ff4fc8';
        ctx.fillRect(630, 190, 6, 2);
        ctx.fillStyle = '#4be4ff';
        ctx.fillRect(4, 136, 6, 2);
        ctx.globalAlpha = 1;
      }
    }

    drawHpBar(ctx, x, y, w, h, f, flip) {
      const n = 20;
      const gap = 1;
      const segW = (w - (n - 1) * gap) / n;

      ctx.fillStyle = '#03060d';
      ctx.fillRect(x - 2, y - 2, w + 4, h + 4);
      ctx.fillStyle = '#0b1526';
      ctx.fillRect(x, y, w, h);

      const ratio = MECHA.clamp(f.hp / CFG.MAX_HP, 0, 1);
      const ratioDelayed = MECHA.clamp(f.hpDisplay / CFG.MAX_HP, 0, 1);
      const color = ratio > 0.5 ? '#63ffa0' : ratio > 0.25 ? '#ffd166' : '#ff5566';

      for (let i = 0; i < n; i++) {
        const index = flip ? n - 1 - i : i;
        const v = (index + 0.5) / n;
        const sx = Math.round(x + i * (segW + gap));
        const ex = Math.round(x + i * (segW + gap) + segW);
        const sw = ex - sx;

        if (v <= ratioDelayed) {
          ctx.fillStyle = 'rgba(255,164,92,0.55)';
          ctx.fillRect(sx, y, sw, h);
        }
        if (v <= ratio) {
          ctx.fillStyle = color;
          ctx.fillRect(sx, y, sw, h);
          ctx.fillStyle = 'rgba(255,255,255,0.22)';
          ctx.fillRect(sx, y, sw, 2);
        }
      }

      // 玩家标识边线
      ctx.fillStyle = f.colors.shell;
      ctx.fillRect(x - 2, y - 2, 10, 1);
    }

    drawGuardBar(ctx, x, y, w, h, f, flip) {
      ctx.fillStyle = '#03060d';
      ctx.fillRect(x - 1, y - 1, w + 2, h + 2);
      ctx.fillStyle = '#0b1526';
      ctx.fillRect(x, y, w, h);

      const ratio = MECHA.clamp(f.guard / CFG.GUARD.max, 0, 1);
      const ww = Math.round(w * ratio);
      const low = ratio <= 0.22;
      ctx.fillStyle = low ? (Math.floor(f.animT * 14) % 2 === 0 ? '#ff5566' : '#ffb0b0') : f.colors.shell;
      if (flip) ctx.fillRect(x + w - ww, y, ww, h);
      else ctx.fillRect(x, y, ww, h);
    }

    drawPips(ctx, x, y, score, color, flip, total) {
      const size = 7;
      const step = size + 4;
      for (let i = 0; i < total; i++) {
        const sx = flip ? x - (i + 1) * step + 4 : x + i * step;
        ctx.fillStyle = '#03060d';
        ctx.fillRect(sx - 1, y - 1, size + 2, size + 2);
        ctx.fillStyle = i < score ? color : '#111c30';
        ctx.fillRect(sx, y, size, size);
        if (i < score) {
          ctx.fillStyle = 'rgba(255,255,255,0.45)';
          ctx.fillRect(sx, y, size, 2);
        }
      }
    }

    drawHud(ctx, game) {
      const p1 = game.fighters[0];
      const p2 = game.fighters[1];

      ctx.fillStyle = 'rgba(3,6,14,0.78)';
      ctx.fillRect(0, 0, CFG.W, 52);
      ctx.fillStyle = '#13203a';
      ctx.fillRect(0, 50, CFG.W, 1);
      ctx.globalAlpha = 0.5;
      ctx.fillStyle = '#4be4ff';
      ctx.fillRect(0, 51, CFG.W, 1);
      ctx.globalAlpha = 1;

      this.drawHpBar(ctx, 20, 15, 246, 12, p1, false);
      this.drawHpBar(ctx, 374, 15, 246, 12, p2, true);

      MECHA.drawText(ctx, 'P1 ' + p1.name, 20, 3, 1, p1.colors.shell, { shadow: '#04070e' });
      MECHA.drawText(ctx, 'P2 ' + p2.name, 620, 3, 1, p2.colors.shell, { align: 'right', shadow: '#04070e' });

      this.drawGuardBar(ctx, 20, 31, 120, 4, p1, false);
      this.drawGuardBar(ctx, 620, 31, 120, 4, p2, true);

      this.drawPips(ctx, 20, 41, game.score[0], p1.colors.shell, false, CFG.WINS_NEEDED);
      this.drawPips(ctx, 620, 41, game.score[1], p2.colors.shell, true, CFG.WINS_NEEDED);

      // 计时器
      ctx.fillStyle = '#03060d';
      ctx.fillRect(291, 3, 58, 40);
      ctx.fillStyle = '#0b1424';
      ctx.fillRect(292, 4, 56, 38);
      ctx.globalAlpha = 0.45;
      ctx.fillStyle = p1.colors.shell;
      ctx.fillRect(292, 4, 18, 1);
      ctx.fillStyle = p2.colors.shell;
      ctx.fillRect(330, 4, 18, 1);
      ctx.globalAlpha = 1;

      const secs = Math.max(0, Math.ceil(game.timer));
      const low = secs <= 10;
      const blink = low && Math.floor(game.time * 5) % 2 === 0;
      const timerColor = low ? (blink ? '#ff5566' : '#ffd4d4') : '#e6f9ff';
      MECHA.drawText(ctx, String(secs).padStart(2, '0'), 320, 10, 3, timerColor, {
        align: 'center', shadow: '#04070e'
      });
      MECHA.drawText(ctx, 'R' + game.round, 320, 33, 1, '#7d92b4', {
        align: 'center', shadow: '#04070e'
      });
    }

    drawBanners(ctx, game) {
      let text = '';
      let sub = '';
      let color = '#ffffff';
      let scale = 4;
      let alpha = 1;

      if (game.state === 'intro') {
        if (game.introT < 0.82) {
          text = 'ROUND ' + game.round;
          color = '#eafcff';
          scale = 4;
          alpha = Math.min(1, game.introT / 0.12);
        } else {
          text = 'FIGHT!';
          color = '#ffd166';
          scale = 5;
          alpha = Math.min(1, (game.introT - 0.82) / 0.1);
        }
      } else if (game.state === 'roundEnd') {
        text = game.endReason === 'ko' ? 'K.O.' : 'TIME UP';
        color = '#ff5566';
        scale = text === 'K.O.' ? 6 : 4;
        sub = game.roundWinner >= 0
          ? game.fighters[game.roundWinner].name + ' WINS ROUND'
          : 'DRAW ROUND';
        alpha = Math.min(1, game.endT / 0.15);
      }

      if (!text) return;

      ctx.globalAlpha = alpha;
      ctx.fillStyle = 'rgba(3,6,14,0.58)';
      ctx.fillRect(0, 136, CFG.W, 82);
      ctx.fillStyle = 'rgba(255,79,200,0.35)';
      ctx.fillRect(0, 136, CFG.W, 1);
      ctx.fillStyle = 'rgba(75,228,255,0.35)';
      ctx.fillRect(0, 217, CFG.W, 1);

      const textY = 176 - (7 * scale) / 2;
      MECHA.drawText(ctx, text, 320, Math.round(textY), scale, color, {
        align: 'center',
        shadow: text === 'K.O.' ? '#4a0b12' : '#04070e',
        shadowOffset: 3
      });

      if (sub) {
        MECHA.drawText(ctx, sub, 320, 198, 1, '#b9d3ee', {
          align: 'center', shadow: '#04070e'
        });
      }
      ctx.globalAlpha = 1;
    }

    render(game) {
      const ctx = this.ctx;
      ctx.save();
      ctx.setTransform(1, 0, 0, 1, 0, 0);
      ctx.imageSmoothingEnabled = false;
      ctx.clearRect(0, 0, CFG.W, CFG.H);

      ctx.drawImage(this.bg, 0, 0);
      this.drawAmbient(ctx, game.time, game.state);

      const shake = game.effects.getShakeOffset();
      ctx.save();
      ctx.translate(shake.x, shake.y);

      MECHA.drawMecha(ctx, game.fighters[0], game.time);
      MECHA.drawMecha(ctx, game.fighters[1], game.time);
      game.effects.draw(ctx);

      ctx.restore();

      if (game.state !== 'menu') {
        this.drawHud(ctx, game);
      }
      this.drawBanners(ctx, game);

      const flash = game.effects.getFlashAlpha();
      if (flash > 0) {
        ctx.globalAlpha = flash;
        ctx.fillStyle = game.effects.flashColor;
        ctx.fillRect(0, 0, CFG.W, CFG.H);
        ctx.globalAlpha = 1;
      }

      ctx.restore();
    }
  };
})();
