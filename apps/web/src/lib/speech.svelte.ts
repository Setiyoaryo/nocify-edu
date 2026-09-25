// Audio instructions via the browser's speech synthesis (§38). No audio files
// are shipped, so there is nothing to license; speech only starts from an
// explicit tap, never autoplay, and the mute setting is remembered per device.

const KEY = 'eduplay.muted';

function readMuted(): boolean {
	try {
		return localStorage.getItem(KEY) === '1';
	} catch {
		return false;
	}
}

class Speech {
	muted = $state(typeof window !== 'undefined' ? readMuted() : false);
	speaking = $state(false);

	get supported() {
		return typeof window !== 'undefined' && 'speechSynthesis' in window;
	}

	toggleMute() {
		this.muted = !this.muted;
		try {
			localStorage.setItem(KEY, this.muted ? '1' : '0');
		} catch {}
		if (this.muted) this.stop();
	}

	say(text: string) {
		if (!this.supported || this.muted || !text) return;
		const synth = window.speechSynthesis;
		synth.cancel();
		const u = new SpeechSynthesisUtterance(text);
		u.lang = 'id-ID';
		u.rate = 0.92;
		const voice = synth.getVoices().find((v) => v.lang.toLowerCase().startsWith('id'));
		if (voice) u.voice = voice;
		u.onstart = () => (this.speaking = true);
		u.onend = u.onerror = () => (this.speaking = false);
		synth.speak(u);
	}

	stop() {
		if (this.supported) window.speechSynthesis.cancel();
		this.speaking = false;
	}
}

export const speech = new Speech();

/**
 * Short synthesized chime for correct answers (§10 micro-feedback). Generated
 * with Web Audio, so no sound file or license is involved. Silent when muted.
 * Wrong answers get no sound on purpose: feedback should never shame.
 */
export function chime() {
	if (speech.muted || typeof window === 'undefined' || !('AudioContext' in window)) return;
	try {
		const ctx = new AudioContext();
		[659.25, 987.77].forEach((freq, i) => {
			const o = ctx.createOscillator();
			const g = ctx.createGain();
			o.type = 'sine';
			o.frequency.value = freq;
			const t = ctx.currentTime + i * 0.11;
			g.gain.setValueAtTime(0.0001, t);
			g.gain.exponentialRampToValueAtTime(0.18, t + 0.02);
			g.gain.exponentialRampToValueAtTime(0.0001, t + 0.35);
			o.connect(g).connect(ctx.destination);
			o.start(t);
			o.stop(t + 0.4);
		});
		setTimeout(() => ctx.close(), 800);
	} catch {}
}
