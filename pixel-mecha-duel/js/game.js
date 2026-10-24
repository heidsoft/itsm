/* ============================================================
   游戏主流程：回合状态机、判定、UI 绑定、主循环
   ============================================================ */

(function () {
  const CFG = MECHA.CFG;
  const NEUTRAL = { left: false, right: false, attack: false, guard: false };

  class Game {
    constructor(canvas) {
      this.canvas = canvas;
      this.renderer = new MECHA.Renderer(canvas);
      this.input = new MECHA.Input();
      this.effects = new MECHA.Effects();
      this.audio = MECHA.Audio;
      this.world = { effects: this.effects, audio: this.audio };

      this.fighters = [
        new MECHA.Fighter(0, MECHA.P1_COLORS, 210),
        new MECHA.Fighter(1, MECHA.P2_COLORS, 430)
      ];

      this.mode = 'ai';
      this.ai = null;
      this.state = 'menu';
      this.time = 0;
      this.round = 1;
      this.timer = CFG.ROUND_TIME;
      this.score = [0, 0];
      this.introT = 0;
      this.endT = 0;
      this.hitstop = 0;
      this.roundWinner = -1;
      this.matchWinner = -1;
      this.endReason = '';
      this.matchOver = false;
      this.pausedFrom = null;
      this._fightSound = false;

      this.ui = {
        menu: document.getElementById('menu'),
        result: document.getElementById('result'),
        pause: document.getElementById('pause'),
        resultTitle: document.getElementById('result-title'),
        resultLine: document.getElementById('result-line'),
        resultScore: document.getElementById('result-score'),
        status: document.getElementById('status-text'),
        statusDot: document.querySelector('.status-dot'),
        sound: document.getElementById('btn-sound'),
        pauseBtn: document.getElementById('btn-pause')
      };

      this.bindUI();

      this.lastTs = 0;
      this.frame = this.frame.bind(this);
      requestAnimationFrame(this.frame);
    }

    bindUI() {
      const self = this;

      document.querySelectorAll('[data-mode]').forEach(function (btn) {
        btn.addEventListener('click', function () {
          self.audio.select();
          self.startMatch(btn.getAttribute('data-mode'));
        });
      });

      document.getElementById('btn-rematch').addEventListener('click', function () {
        self.audio.select();
        self.startMatch(self.mode);
      });
      document.getElementById('btn-title').addEventListener('click', function () {
        self.audio.select();
        self.toMenu();
      });
      document.getElementById('btn-resume').addEventListener('click', function () {
        self.audio.select();
        self.resume();
      });
      document.getElementById('btn-restart').addEventListener('click', function () {
        self.audio.select();
        self.resume();
        self.startMatch(self.mode);
      });
      document.getElementById('btn-quit').addEventListener('click', function () {
        self.audio.select();
        self.toMenu();
      });
      document.getElementById('btn-sound').addEventListener('click', function () {
        self.toggleSound();
      });
      document.getElementById('btn-pause').addEventListener('click', function () {
        self.togglePause();
      });

      window.addEventListener('keydown', function (e) {
        if (e.repeat) return;
        if (e.code === 'KeyP') {
          e.preventDefault();
          self.togglePause();
        } else if (e.code === 'KeyM') {
          e.preventDefault();
          self.toggleSound();
        }
      });

      document.addEventListener('visibilitychange', function () {
        if (document.hidden && ['fight', 'intro', 'roundEnd'].indexOf(self.state) >= 0) {
          self.pause();
        }
      });
    }

    /* ---------------- 状态切换 ---------------- */

    startMatch(mode) {
      this.audio.init();
      this.audio.setEnabled(this.audio.enabled);
      if (document.activeElement && document.activeElement.blur) document.activeElement.blur();

      this.mode = mode === 'pvp' ? 'pvp' : 'ai';
      this.ai = this.mode === 'ai' ? new MECHA.AI(this.fighters[1], this.fighters[0]) : null;
      this.score = [0, 0];
      this.round = 1;
      this.matchOver = false;
      this.matchWinner = -1;

      this.ui.menu.classList.add('hidden');
      this.ui.result.classList.add('hidden');
      this.ui.pause.classList.add('hidden');

      this.startRound();
      this.setStatus((this.mode === 'ai' ? '人机对战' : '双人对战') + ' · 三回合决胜', true);
    }

    startRound() {
      this.fighters[0].reset();
      this.fighters[1].reset();
      this.effects.clear();

      if (this.ai) this.ai.reset();

      this.timer = CFG.ROUND_TIME;
      this.introT = 0;
      this.endT = 0;
      this.hitstop = 0;
      this.roundWinner = -1;
      this.endReason = '';
      this.state = 'intro';
      this._fightSound = false;
      this.audio.beep(440);
      this.setStatus('第 ' + this.round + ' 回合 · 准备', true);
    }

    toMenu() {
      this.state = 'menu';
      this.pausedFrom = null;
      this.ai = null;
      this.score = [0, 0];
      this.round = 1;
      this.timer = CFG.ROUND_TIME;
      this.hitstop = 0;
      this.effects.clear();

      this.fighters[0].reset();
      this.fighters[1].reset();
      this.fighters[0].x = 226;
      this.fighters[1].x = 414;

      this.ui.menu.classList.remove('hidden');
      this.ui.result.classList.add('hidden');
      this.ui.pause.classList.add('hidden');
      this.setStatus('待机中 · 选择对战模式', false);
    }

    pause() {
      if (['intro', 'fight', 'roundEnd'].indexOf(this.state) < 0) return;
      this.pausedFrom = this.state;
      this.state = 'paused';
      this.ui.pause.classList.remove('hidden');
      this.audio.pause();
      this.setStatus('已暂停', false);
    }

    resume() {
      if (this.state !== 'paused') return;
      this.state = this.pausedFrom || 'fight';
      this.pausedFrom = null;
      this.ui.pause.classList.add('hidden');
      this.setStatus(this.state === 'roundEnd' ? '回合结束' : '战斗中', true);
    }

    togglePause() {
      if (this.state === 'paused') this.resume();
      else this.pause();
    }

    toggleSound() {
      this.audio.init();
      const on = !this.audio.enabled;
      this.audio.setEnabled(on);
      this.ui.sound.textContent = on ? '♪ 音效 ON' : '♪ 音效 OFF';
      this.ui.sound.setAttribute('aria-pressed', String(on));
      if (on) this.audio.select();
    }

    setStatus(text, busy) {
      this.ui.status.textContent = text;
      this.ui.statusDot.classList.toggle('is-busy', !!busy);
    }

    /* ---------------- 回合结算 ---------------- */

    endRound(reason) {
      if (this.state !== 'fight') return;

      const p1 = this.fighters[0];
      const p2 = this.fighters[1];
      let winner = -1;

      if (p1.hp <= 0 && p2.hp <= 0) winner = -1;
      else if (p2.hp <= 0) winner = 0;
      else if (p1.hp <= 0) winner = 1;
      else if (reason === 'time') winner = p1.hp > p2.hp ? 0 : p2.hp > p1.hp ? 1 : -1;

      if (winner >= 0) this.score[winner]++;

      this.roundWinner = winner;
      this.endReason = reason;
      this.state = 'roundEnd';
      this.endT = 0;
      this.matchOver =
        this.score[0] >= CFG.WINS_NEEDED ||
        this.score[1] >= CFG.WINS_NEEDED ||
        this.round >= CFG.MAX_ROUNDS;

      if (reason === 'time') this.audio.beep(220);
      this.setStatus(reason === 'ko' ? 'K.O.' : '时间到', false);
    }

    endMatch() {
      const p1 = this.fighters[0];
      const p2 = this.fighters[1];
      let winner = -1;
      if (this.score[0] > this.score[1]) winner = 0;
      else if (this.score[1] > this.score[0]) winner = 1;

      this.matchWinner = winner;
      this.state = 'matchEnd';

      const title = this.ui.resultTitle;
      const line = this.ui.resultLine;
      const scoreEl = this.ui.resultScore;

      title.textContent = winner >= 0 ? 'VICTORY' : 'DRAW';
      if (winner >= 0) {
        const col = winner === 0 ? '#4be4ff' : '#ff6a4d';
        title.style.color = col;
        title.style.textShadow =
          '3px 0 rgba(255,79,200,0.65), -3px 0 rgba(75,228,255,0.65), 0 0 38px ' + col + '99, 0 7px 0 rgba(3,8,18,0.85)';
        line.textContent = this.fighters[winner].name + ' 获胜';
        line.style.color = col;
        this.audio.win();
      } else {
        title.style.color = '#ffd166';
        title.style.textShadow =
          '3px 0 rgba(255,79,200,0.55), -3px 0 rgba(75,228,255,0.55), 0 0 38px rgba(255,209,102,0.4), 0 7px 0 rgba(3,8,18,0.85)';
        line.textContent = '双方平局 · 势均力敌';
        line.style.color = '#ffd166';
        this.audio.beep(330);
      }

      scoreEl.textContent = this.score[0] + ' : ' + this.score[1];
      this.ui.result.classList.remove('hidden');
      this.setStatus('比赛结束', false);
    }

    /* ---------------- 战斗判定 ---------------- */

    separateFighters() {
      const p1 = this.fighters[0];
      const p2 = this.fighters[1];
      const minDist = 34;
      const dx = p2.x - p1.x;
      const overlap = minDist - Math.abs(dx);
      if (overlap > 0) {
        const dir = dx >= 0 ? 1 : -1;
        const push = overlap / 2;
        p1.x -= dir * push;
        p2.x += dir * push;
        p1.clampX();
        p2.clampX();
      }
    }

    resolveHits() {
      this.tryHit(this.fighters[0], this.fighters[1]);
      this.tryHit(this.fighters[1], this.fighters[0]);
    }

    tryHit(attacker, defender) {
      const hitbox = attacker.getHitbox();
      if (!hitbox || defender.state === 'ko') return;

      const hurt = defender.getHurtbox();
      if (hitbox.x2 < hurt.x1 || hitbox.x1 > hurt.x2) return;

      attacker.attackHasHit = true;
      const result = defender.takeHit(CFG.ATTACK, attacker, this.world);
      if (!result) return;

      this.hitstop = result.blocked ? 0.03 : result.counter ? 0.085 : 0.055;
    }

    /* ---------------- 主循环 ---------------- */

    update(dt) {
      this.time += dt;

      if (this.state === 'paused') return;

      if (this.state === 'menu') {
        this.fighters[0].update(dt, NEUTRAL, this.fighters[1], this.world);
        this.fighters[1].update(dt, NEUTRAL, this.fighters[0], this.world);
        this.effects.update(dt);
        return;
      }

      if (this.state === 'intro') {
        this.introT += dt;
        this.fighters[0].update(dt, NEUTRAL, this.fighters[1], this.world);
        this.fighters[1].update(dt, NEUTRAL, this.fighters[0], this.world);
        this.effects.update(dt);

        if (!this._fightSound && this.introT >= 0.82) {
          this._fightSound = true;
          this.audio.fight();
          this.setStatus('FIGHT!', true);
        }
        if (this.introT >= 1.5) {
          this.state = 'fight';
          this.setStatus('战斗中', true);
        }
        return;
      }

      if (this.state === 'fight') {
        if (this.hitstop > 0) {
          this.hitstop -= dt;
          this.effects.update(dt * 0.35);
          return;
        }

        const p1Controls = this.input.P1;
        const p2Controls = this.mode === 'ai' ? this.ai.update(dt) : this.input.P2;

        this.fighters[0].update(dt, p1Controls, this.fighters[1], this.world);
        this.fighters[1].update(dt, p2Controls, this.fighters[0], this.world);

        this.separateFighters();
        this.resolveHits();
        this.effects.update(dt);

        if (this.fighters[0].hp <= 0 || this.fighters[1].hp <= 0) {
          this.endRound('ko');
          return;
        }

        this.timer -= dt;
        if (this.timer <= 0) {
          this.timer = 0;
          this.endRound('time');
        }
        return;
      }

      if (this.state === 'roundEnd') {
        this.endT += dt;
        this.fighters[0].update(dt, NEUTRAL, this.fighters[1], this.world);
        this.fighters[1].update(dt, NEUTRAL, this.fighters[0], this.world);
        this.effects.update(dt);

        if (this.endT >= 2.5) {
          if (this.matchOver) this.endMatch();
          else {
            this.round++;
            this.startRound();
          }
        }
        return;
      }

      if (this.state === 'matchEnd') {
        this.fighters[0].update(dt, NEUTRAL, this.fighters[1], this.world);
        this.fighters[1].update(dt, NEUTRAL, this.fighters[0], this.world);
        this.effects.update(dt);
      }
    }

    frame(ts) {
      if (!this.lastTs) this.lastTs = ts;
      let dt = (ts - this.lastTs) / 1000;
      this.lastTs = ts;
      if (dt > 0.05) dt = 0.05;

      this.update(dt);
      this.renderer.render(this);
      this.input.endFrame();

      requestAnimationFrame(this.frame);
    }
  }

  MECHA.Game = Game;

  function boot() {
    const canvas = document.getElementById('game');
    if (canvas) new Game(canvas);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', boot);
  } else {
    boot();
  }
})();
