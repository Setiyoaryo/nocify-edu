// Voice over audio player & speech synthesis (§38).
// Uses recorded MP3 voiceovers for game feedbacks with Web Speech fallback for dynamic text.

const KEY = 'eduplay.muted';

function readMuted(): boolean {
	try {
		return localStorage.getItem(KEY) === '1';
	} catch {
		return false;
	}
}

function cleanText(t: string): string {
	return t
		.toLowerCase()
		.trim()
		.replace(/[.,!?;:"'~]/g, '')
		.replace(/-/g, ' ')
		.replace(/\s+/g, ' ');
}

const AUDIO_VOICEOVERS: Record<string, string> = {
	'hebat jawabanmu benar': '/sounds/hebat-jawabanmu-benar.mp3',
	'keren kamu berhasil': '/sounds/keren-kamu-berhasil.mp3',
	'tepat sekali terus semangat': '/sounds/tepat-sekali-terus-semangat.mp3',
	'bagus kamu berhasil setelah mencoba lagi': '/sounds/bagus-kamu-berhasil-setelah-mencoba-lagi.mp3',
	'hampir benar coba lihat lagi petunjuknya': '/sounds/hampir-benar-coba-lihat-lagi-petunjuknya.mp3',
	'tidak apa apa ini jawaban yang benar yuk lihat penjelasannya': '/sounds/tidak-apa-apa-ini-jawaban-yang-benar.mp3'
};

class Speech {
	muted = $state(typeof window !== 'undefined' ? readMuted() : false);
	speaking = $state(false);
	private currentAudio: HTMLAudioElement | null = null;

	get supported() {
		return typeof window !== 'undefined' && ('Audio' in window || 'speechSynthesis' in window);
	}

	toggleMute() {
		this.muted = !this.muted;
		try {
			localStorage.setItem(KEY, this.muted ? '1' : '0');
		} catch {}
		if (this.muted) this.stop();
	}

	say(text: string) {
		if (this.muted || !text || typeof window === 'undefined') return;
		this.stop();

		const norm = cleanText(text);
		const soundSrc = AUDIO_VOICEOVERS[norm];

		if (soundSrc) {
			const audio = new Audio(soundSrc);
			this.currentAudio = audio;
			this.speaking = true;
			audio.onended = () => {
				if (this.currentAudio === audio) {
					this.speaking = false;
					this.currentAudio = null;
				}
			};
			audio.onerror = () => {
				if (this.currentAudio === audio) {
					this.speaking = false;
					this.currentAudio = null;
				}
			};
			audio.play().catch(() => {
				if ('speechSynthesis' in window) {
					this.speakSynthesis(text);
				} else {
					this.speaking = false;
				}
			});
			return;
		}

		if ('speechSynthesis' in window) {
			this.speakSynthesis(text);
		}
	}

	private speakSynthesis(text: string) {
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
		if (this.currentAudio) {
			this.currentAudio.pause();
			this.currentAudio.currentTime = 0;
			this.currentAudio = null;
		}
		if (typeof window !== 'undefined' && 'speechSynthesis' in window) {
			window.speechSynthesis.cancel();
		}
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
