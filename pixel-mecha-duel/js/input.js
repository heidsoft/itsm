/* ============================================================
   键盘输入：持续按键 + 单帧触发
   ============================================================ */

MECHA.Input = class Input {
  constructor() {
    this.down = new Set();
    this.pressed = new Set();

    window.addEventListener('keydown', (e) => {
      // 方向键 / 空格默认会滚动页面
      if (['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Space'].includes(e.code)) {
        e.preventDefault();
      }
      if (e.repeat) return;
      this.down.add(e.code);
      this.pressed.add(e.code);
    });

    window.addEventListener('keyup', (e) => {
      this.down.delete(e.code);
    });

    window.addEventListener('blur', () => {
      this.down.clear();
      this.pressed.clear();
    });
  }

  isDown(code) {
    return this.down.has(code);
  }

  anyDown(codes) {
    for (let i = 0; i < codes.length; i++) {
      if (this.down.has(codes[i])) return true;
    }
    return false;
  }

  wasPressed(code) {
    return this.pressed.has(code);
  }

  anyPressed(codes) {
    for (let i = 0; i < codes.length; i++) {
      if (this.pressed.has(codes[i])) return true;
    }
    return false;
  }

  /** 玩家 1：A / D 移动，J 攻击，K 防御 */
  get P1() {
    return {
      left: this.isDown('KeyA'),
      right: this.isDown('KeyD'),
      attack: this.wasPressed('KeyJ'),
      guard: this.isDown('KeyK')
    };
  }

  /** 玩家 2：← / → 移动，1 攻击，2 防御（含小键盘） */
  get P2() {
    return {
      left: this.anyDown(['ArrowLeft']),
      right: this.anyDown(['ArrowRight']),
      attack: this.anyPressed(['Digit1', 'Numpad1']),
      guard: this.anyDown(['Digit2', 'Numpad2'])
    };
  }

  endFrame() {
    this.pressed.clear();
  }
};
