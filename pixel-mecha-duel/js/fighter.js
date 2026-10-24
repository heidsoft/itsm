/* ============================================================
   机甲角色：移动 / 攻击 / 防御 / 受击 / 击破
   所有状态时间与判定均为权威逻辑，渲染层只读取状态。
   ============================================================ */

MECHA.Fighter = class Fighter {
  constructor(id, colors, spawnX) {
    this.id = id;
    this.colors = colors;
    this.name = colors.name;
    this.spawnX = spawnX;
    this.reset();
  }

  reset() {
    const C = MECHA.CFG;
    this.x = this.spawnX;
    this.vx = 0;
    this.facing = this.id === 0 ? 1 : -1;

    this.hp = C.MAX_HP;
    this.hpDisplay = C.MAX_HP;
    this.guard = C.GUARD.max;
    this.guardDisplay = C.GUARD.max;

    this.state = 'idle';
    this.stateTime = 0;
    this.hitDuration = 0;
    this.attackHasHit = false;
    this.cooldown = 0;
    this.hitFlash = 0;

    this.animT = Math.random() * 2;
    this.walkPhase = 0;
    this.stepTimer = 0;
    this.koTime = 0;
  }

  clampX() {
    const C = MECHA.CFG;
    this.x = MECHA.clamp(this.x, C.LEFT, C.RIGHT);
  }

  getAttackPhase() {
    if (this.state !== 'attack') return null;
    const A = MECHA.CFG.ATTACK;
    if (this.stateTime < A.startup) return 'startup';
    if (this.stateTime < A.startup + A.active) return 'active';
    return 'recovery';
  }

  getHitbox() {
    if (this.state !== 'attack' || this.attackHasHit) return null;
    if (this.getAttackPhase() !== 'active') return null;
    const C = MECHA.CFG;
    const A = C.ATTACK;
    const x1 = this.facing === 1 ? this.x + A.reach : this.x - A.reach - A.range;
    return { x1: x1, x2: x1 + A.range, y1: C.GROUND - 62, y2: C.GROUND };
  }

  getHurtbox() {
    const C = MECHA.CFG;
    return { x1: this.x - 16, x2: this.x + 16, y1: C.GROUND - 68, y2: C.GROUND };
  }

  update(dt, controls, opponent, world) {
    const C = MECHA.CFG;
    const A = C.ATTACK;

    this.animT += dt;
    this.hitFlash = Math.max(0, this.hitFlash - dt);
    this.cooldown = Math.max(0, this.cooldown - dt);

    // 掉血条 / 护盾条平滑跟随
    this.hpDisplay += (this.hp - this.hpDisplay) * Math.min(1, dt * 7);
    this.guardDisplay += (this.guard - this.guardDisplay) * Math.min(1, dt * 14);

    if (this.state === 'ko') {
      this.koTime += dt;
      this.vx *= Math.pow(0.002, dt);
      this.x += this.vx * dt;
      this.clampX();
      return;
    }

    // 受击硬直 / 破防硬直
    if (this.state === 'hit' || this.state === 'guardBreak') {
      this.stateTime += dt;
      this.vx *= Math.pow(0.0015, dt);
      this.x += this.vx * dt;
      this.clampX();
      const duration = this.state === 'hit' ? this.hitDuration : C.GUARD.breakStun;
      if (this.stateTime >= duration) {
        this.state = 'idle';
        this.stateTime = 0;
        this.vx = 0;
      }
      return;
    }

    // 自动面向对手
    if (this.state !== 'attack') {
      this.facing = opponent.x < this.x ? -1 : 1;
    }

    // 攻击动作：起手前冲 -> 回收
    if (this.state === 'attack') {
      this.stateTime += dt;
      if (this.stateTime < A.startup) {
        this.vx = this.facing * A.lunge;
      } else if (this.stateTime < A.startup + A.active) {
        this.vx = this.facing * A.lunge * 0.22;
      } else {
        this.vx *= Math.pow(0.001, dt);
      }
      this.x += this.vx * dt;
      this.clampX();

      if (this.stateTime >= A.startup + A.active + A.recovery) {
        this.state = 'idle';
        this.stateTime = 0;
        this.vx = 0;
      }
      return;
    }

    // 防御姿态
    const wantGuard = !!controls.guard && this.guard > 0;
    if (wantGuard) {
      if (this.state !== 'guard') {
        this.state = 'guard';
        this.stateTime = 0;
        this.vx = 0;
      }
    } else if (this.state === 'guard') {
      this.state = 'idle';
      this.stateTime = 0;
    }

    if (this.state === 'guard') {
      this.guard = Math.max(0, this.guard - C.GUARD.drain * dt);
      if (this.guard <= 0) {
        this.breakGuard(world);
        return;
      }
    } else {
      this.guard = Math.min(C.GUARD.max, this.guard + C.GUARD.regen * dt);
    }

    // 攻击输入（防御中可直接出拳）
    if (controls.attack && this.cooldown <= 0 &&
        (this.state === 'idle' || this.state === 'walk' || this.state === 'guard')) {
      this.startAttack(world);
      return;
    }

    // 水平移动
    let dir = (controls.right ? 1 : 0) - (controls.left ? 1 : 0);
    if (this.state === 'guard') dir *= C.GUARD_MOVE_MUL;

    const targetVx = dir * C.WALK_SPEED;
    const accel = dir !== 0 ? C.ACCEL : C.FRICTION;
    if (this.vx < targetVx) this.vx = Math.min(targetVx, this.vx + accel * dt);
    else if (this.vx > targetVx) this.vx = Math.max(targetVx, this.vx - accel * dt);

    this.x += this.vx * dt;
    this.clampX();

    if (this.state !== 'guard') {
      if (Math.abs(this.vx) > 8) {
        this.state = 'walk';
        this.walkPhase += dt * (7 + Math.abs(this.vx) / 24);
        this.stepTimer -= dt;
        if (this.stepTimer <= 0) {
          this.stepTimer = 0.13;
          world.effects.dust(this.x - this.facing * 5, C.GROUND - 1, this.facing, '#37445c');
        }
      } else {
        this.state = 'idle';
        this.walkPhase = 0;
      }
    }
  }

  startAttack(world) {
    const C = MECHA.CFG;
    this.state = 'attack';
    this.stateTime = 0;
    this.attackHasHit = false;
    this.cooldown = C.ATTACK.cooldown;
    this.walkPhase = 0;
    this.vx = this.facing * C.ATTACK.lunge;
    if (world && world.audio) world.audio.whiff();
  }

  breakGuard(world) {
    const C = MECHA.CFG;
    this.state = 'guardBreak';
    this.stateTime = 0;
    this.guard = 0;
    this.vx = -this.facing * C.GUARD.breakKnockback;
    if (world) {
      world.effects.guardBreak(this.x + this.facing * 12, C.GROUND - 34, this.colors.shell);
      world.effects.text(this.x, C.GROUND - 82, 'BREAK', '#ffd166', 1);
      world.audio.guardBreak();
    }
  }

  /**
   * 受到一次攻击。
   * @returns {{blocked:boolean, damage:number, counter:boolean}|null}
   */
  takeHit(attack, attacker, world) {
    const C = MECHA.CFG;
    if (this.state === 'ko') return null;

    const fromFront = (attacker.x - this.x) * this.facing > 0;
    const blocking = this.state === 'guard' && fromFront;

    if (blocking) {
      const chip = attack.chip;
      this.hp = Math.max(0, this.hp - chip);
      this.guard = Math.max(0, this.guard - attack.guardCost);
      this.vx = -this.facing * attack.knockback * 0.32;

      world.effects.block(this.x + this.facing * 16, C.GROUND - 40, this.colors.shell);
      world.effects.text(this.x, C.GROUND - 78, 'BLOCK', '#bff6ff', 1);
      world.audio.block();

      if (this.guard <= 0) this.breakGuard(world);
      if (this.hp <= 0) this.knockOut(attacker, world);
      return { blocked: true, damage: chip, counter: false };
    }

    const counter = this.state === 'attack' && this.getAttackPhase() === 'startup';
    let damage = attack.damage;
    if (counter) damage = Math.round(damage * attack.counterMul);

    this.hp = Math.max(0, this.hp - damage);
    this.hitFlash = 0.16;
    this.state = 'hit';
    this.stateTime = 0;
    this.hitDuration = counter ? attack.counterHitstun : attack.hitstun;
    this.vx = (this.x < attacker.x ? -1 : 1) * attack.knockback * (counter ? 1.35 : 1);
    this.attackHasHit = false;

    const dir = attacker.x < this.x ? 1 : -1;
    world.effects.hit(this.x + dir * 14, C.GROUND - 40, dir, this.colors.shell, counter);
    world.effects.text(this.x, C.GROUND - 84, (counter ? damage + '!' : String(damage)), counter ? '#ffd166' : '#ffffff', counter ? 2 : 1);
    world.audio.hit(counter);

    if (this.hp <= 0) this.knockOut(attacker, world);

    return { blocked: false, damage: damage, counter: counter };
  }

  knockOut(attacker, world) {
    const C = MECHA.CFG;
    this.hp = 0;
    this.state = 'ko';
    this.stateTime = 0;
    this.koTime = 0;
    this.vx = (this.x < attacker.x ? -1 : 1) * 190;
    world.effects.explode(this.x, C.GROUND - 30, this.colors.shell);
    world.audio.ko();
  }
};
