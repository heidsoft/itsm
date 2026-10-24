/* ============================================================
   8-bit 音效合成（WebAudio，无外部音频文件）
   ============================================================ */

MECHA.Audio = {
  ctx: null,
  master: null,
  enabled: true,
  noiseBuf: null,

  init() {
    if (this.ctx) {
      if (this.ctx.state === 'suspended') this.ctx.resume();
      return;
    }
    const AC = window.AudioContext || window.webkitAudioContext;
    if (!AC) return;
    this.ctx = new AC();
    this.master = this.ctx.createGain();
    this.master.gain.value = 0.5;
    this.master.connect(this.ctx.destination);
    this._buildNoise();
  },

  setEnabled(on) {
    this.enabled = on;
    if (this.master) {
      this.master.gain.setTargetAtTime(on ? 0.5 : 0, this.ctx.currentTime, 0.02);
    }
  },

  _buildNoise() {
    const len = Math.floor(this.ctx.sampleRate * 0.7);
    const buf = this.ctx.createBuffer(1, len, this.ctx.sampleRate);
    const data = buf.getChannelData(0);
    for (let i = 0; i < len; i++) data[i] = Math.random() * 2 - 1;
    this.noiseBuf = buf;
  },

  tone(freq, endFreq, dur, type, vol, delay) {
    if (!this.ctx || !this.enabled) return;
    const t0 = this.ctx.currentTime + (delay || 0);
    const osc = this.ctx.createOscillator();
    const gain = this.ctx.createGain();
    osc.type = type || 'square';
    osc.frequency.setValueAtTime(Math.max(20, freq), t0);
    if (endFreq && endFreq !== freq) {
      osc.frequency.exponentialRampToValueAtTime(Math.max(20, endFreq), t0 + dur);
    }
    gain.gain.setValueAtTime(0.0001, t0);
    gain.gain.linearRampToValueAtTime(vol || 0.14, t0 + 0.006);
    gain.gain.exponentialRampToValueAtTime(0.0001, t0 + dur);
    osc.connect(gain);
    gain.connect(this.master);
    osc.start(t0);
    osc.stop(t0 + dur + 0.03);
  },

  noise(dur, vol, freq, type, delay) {
    if (!this.ctx || !this.enabled || !this.noiseBuf) return;
    const t0 = this.ctx.currentTime + (delay || 0);
    const src = this.ctx.createBufferSource();
    src.buffer = this.noiseBuf;
    const filter = this.ctx.createBiquadFilter();
    filter.type = type || 'lowpass';
    filter.frequency.value = freq || 1200;
    const gain = this.ctx.createGain();
    gain.gain.setValueAtTime(vol || 0.2, t0);
    gain.gain.exponentialRampToValueAtTime(0.0001, t0 + dur);
    src.connect(filter);
    filter.connect(gain);
    gain.connect(this.master);
    src.start(t0);
    src.stop(t0 + dur);
  },

  /* ---------- 游戏音效 ---------- */

  hit(counter) {
    this.tone(counter ? 260 : 170, 60, 0.12, 'sawtooth', counter ? 0.24 : 0.16);
    this.noise(0.09, counter ? 0.34 : 0.22, counter ? 2600 : 1500, 'lowpass');
    if (counter) this.tone(880, 620, 0.14, 'square', 0.1, 0.03);
  },

  block() {
    this.tone(1180, 760, 0.06, 'square', 0.11);
    this.noise(0.05, 0.14, 3800, 'highpass');
  },

  guardBreak() {
    this.tone(520, 90, 0.32, 'sawtooth', 0.2);
    this.noise(0.28, 0.3, 700, 'lowpass');
  },

  whiff() {
    this.noise(0.14, 0.09, 2600, 'highpass');
  },

  ko() {
    this.noise(0.55, 0.42, 620, 'lowpass');
    this.tone(190, 38, 0.6, 'sawtooth', 0.24);
    this.tone(920, 240, 0.22, 'square', 0.1, 0.02);
  },

  beep(freq) {
    this.tone(freq || 660, freq || 660, 0.07, 'square', 0.12);
  },

  fight() {
    this.tone(440, 440, 0.09, 'square', 0.13);
    this.tone(660, 660, 0.09, 'square', 0.13, 0.1);
    this.tone(990, 990, 0.16, 'square', 0.14, 0.2);
  },

  win() {
    const notes = [523, 659, 784, 1046];
    notes.forEach((n, i) => this.tone(n, n, 0.16, 'square', 0.13, i * 0.11));
  },

  select() {
    this.tone(880, 1180, 0.05, 'square', 0.09);
  },

  pause() {
    this.tone(660, 330, 0.12, 'square', 0.08);
  }
};
