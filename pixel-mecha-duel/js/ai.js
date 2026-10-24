/* ============================================================
   电脑对手：轻量状态机（接近 / 出拳 / 格挡 / 后撤）
   ============================================================ */

MECHA.AI = class AI {
  constructor(self, foe) {
    this.self = self;
    this.foe = foe;
    this.reset();
  }

  reset() {
    this.think = 0;
    this.plan = 'wait';
    this.planTime = 0;
    this.attackPulse = 0;
    this.guardHold = 0;
  }

  update(dt) {
    const s = this.self;
    const f = this.foe;
    const controls = { left: false, right: false, guard: false, attack: false };

    if (s.state === 'ko' || s.state === 'hit' || s.state === 'guardBreak') {
      return controls;
    }

    this.attackPulse = Math.max(0, this.attackPulse - dt);
    this.guardHold = Math.max(0, this.guardHold - dt);
    this.planTime -= dt;
    this.think -= dt;

    const dist = Math.abs(f.x - s.x);
    const foePhase = f.getAttackPhase();
    const danger = f.state === 'attack' && dist < 74 && foePhase !== 'recovery';

    if (this.think <= 0) {
      this.think = 0.13 + Math.random() * 0.15;
      this.decide(dist, danger);
    }

    if (this.plan === 'approach') {
      if (f.x < s.x) controls.left = true;
      else controls.right = true;
      // 保持一点距离，不要一直贴脸
      if (dist < 34) {
        controls.left = false;
        controls.right = false;
      }
    } else if (this.plan === 'retreat') {
      if (s.x < 70) {
        controls.right = true;
      } else if (s.x > 570) {
        controls.left = true;
      } else if (f.x < s.x) {
        controls.right = true;
      } else {
        controls.left = true;
      }
    } else if (this.plan === 'guard') {
      if (this.guardHold > 0 && s.guard > 18) controls.guard = true;
      if (s.guard <= 18) this.plan = 'retreat';
    } else if (this.plan === 'attack') {
      if (this.attackPulse > 0) controls.attack = true;
    }

    // 空闲时重新决策，避免长时间站立不动
    if (this.planTime <= 0 && this.plan !== 'attack') {
      this.think = 0;
    }

    return controls;
  }

  decide(dist, danger) {
    const s = this.self;
    const r = Math.random();

    if (danger) {
      if (s.guard > 26 && r < 0.58) {
        this.setPlan('guard', 0.3);
        this.guardHold = 0.34;
        return;
      }
      if (r < 0.8) {
        this.setPlan('retreat', 0.24);
        return;
      }
      if (dist < 56 && s.cooldown <= 0 && s.guard > 14) {
        this.setPlan('attack', 0.2);
        this.attackPulse = 0.05;
        return;
      }
      this.setPlan('approach', 0.24);
      return;
    }

    if (dist < 50) {
      if (s.cooldown <= 0 && r < 0.62) {
        this.setPlan('attack', 0.22);
        this.attackPulse = 0.05;
        return;
      }
      if (r < 0.8 && s.guard > 24) {
        this.setPlan('guard', 0.22);
        this.guardHold = 0.2;
        return;
      }
      this.setPlan('retreat', 0.18);
      return;
    }

    if (dist < 84) {
      if (r < 0.7) this.setPlan('approach', 0.3);
      else this.setPlan('wait', 0.26);
      return;
    }

    if (r < 0.88) this.setPlan('approach', 0.34);
    else this.setPlan('wait', 0.3);
  }

  setPlan(name, time) {
    this.plan = name;
    this.planTime = time;
  }
};
