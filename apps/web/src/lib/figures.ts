// Every pictured thing in EduPlay (question objects, avatars, badges, rewards) is a
// Phosphor icon (MIT) rendered in its "fill" weight: one consistent, licensed family and
// no emoji (see docs/brand-assets.md). The approved question figures come from
// packages/figures/figures.json — the single source of truth shared with the Go API
// (apps/api/internal/questions) and the Python pipeline. A test (figures.test.ts) checks
// this file covers every name there; a Go test checks the same in the other direction.
import type { Component } from 'svelte';
import data from '../../../../packages/figures/figures.json';
import Acorn from 'phosphor-svelte/lib/Acorn';
import Airplane from 'phosphor-svelte/lib/Airplane';
import Avocado from 'phosphor-svelte/lib/Avocado';
import Backpack from 'phosphor-svelte/lib/Backpack';
import Balloon from 'phosphor-svelte/lib/Balloon';
import Basketball from 'phosphor-svelte/lib/Basketball';
import Bed from 'phosphor-svelte/lib/Bed';
import Bell from 'phosphor-svelte/lib/Bell';
import Bicycle from 'phosphor-svelte/lib/Bicycle';
import Bird from 'phosphor-svelte/lib/Bird';
import Boat from 'phosphor-svelte/lib/Boat';
import Book from 'phosphor-svelte/lib/Book';
import BookOpen from 'phosphor-svelte/lib/BookOpen';
import Boot from 'phosphor-svelte/lib/Boot';
import BowlFood from 'phosphor-svelte/lib/BowlFood';
import Bread from 'phosphor-svelte/lib/Bread';
import Bug from 'phosphor-svelte/lib/Bug';
import Bus from 'phosphor-svelte/lib/Bus';
import Butterfly from 'phosphor-svelte/lib/Butterfly';
import Cactus from 'phosphor-svelte/lib/Cactus';
import Cake from 'phosphor-svelte/lib/Cake';
import Calendar from 'phosphor-svelte/lib/Calendar';
import Car from 'phosphor-svelte/lib/Car';
import Carrot from 'phosphor-svelte/lib/Carrot';
import Cat from 'phosphor-svelte/lib/Cat';
import Chalkboard from 'phosphor-svelte/lib/Chalkboard';
import Cheese from 'phosphor-svelte/lib/Cheese';
import Cherries from 'phosphor-svelte/lib/Cherries';
import Circle from 'phosphor-svelte/lib/Circle';
import Clock from 'phosphor-svelte/lib/Clock';
import Cloud from 'phosphor-svelte/lib/Cloud';
import CloudRain from 'phosphor-svelte/lib/CloudRain';
import Compass from 'phosphor-svelte/lib/Compass';
import Cookie from 'phosphor-svelte/lib/Cookie';
import Cow from 'phosphor-svelte/lib/Cow';
import Crown from 'phosphor-svelte/lib/Crown';
import Diamond from 'phosphor-svelte/lib/Diamond';
import Dog from 'phosphor-svelte/lib/Dog';
import Drop from 'phosphor-svelte/lib/Drop';
import Egg from 'phosphor-svelte/lib/Egg';
import Eraser from 'phosphor-svelte/lib/Eraser';
import Fire from 'phosphor-svelte/lib/Fire';
import Fish from 'phosphor-svelte/lib/Fish';
import Flag from 'phosphor-svelte/lib/Flag';
import Flower from 'phosphor-svelte/lib/Flower';
import FlowerTulip from 'phosphor-svelte/lib/FlowerTulip';
import ForkKnife from 'phosphor-svelte/lib/ForkKnife';
import Gift from 'phosphor-svelte/lib/Gift';
import Globe from 'phosphor-svelte/lib/Globe';
import GraduationCap from 'phosphor-svelte/lib/GraduationCap';
import Hamburger from 'phosphor-svelte/lib/Hamburger';
import Hand from 'phosphor-svelte/lib/Hand';
import HandSoap from 'phosphor-svelte/lib/HandSoap';
import Heart from 'phosphor-svelte/lib/Heart';
import Hexagon from 'phosphor-svelte/lib/Hexagon';
import Horse from 'phosphor-svelte/lib/Horse';
import House from 'phosphor-svelte/lib/House';
import IceCream from 'phosphor-svelte/lib/IceCream';
import Leaf from 'phosphor-svelte/lib/Leaf';
import Lightning from 'phosphor-svelte/lib/Lightning';
import MapTrifold from 'phosphor-svelte/lib/MapTrifold';
import Moon from 'phosphor-svelte/lib/Moon';
import Notebook from 'phosphor-svelte/lib/Notebook';
import Orange from 'phosphor-svelte/lib/Orange';
import PaintBrush from 'phosphor-svelte/lib/PaintBrush';
import Palette from 'phosphor-svelte/lib/Palette';
import Paperclip from 'phosphor-svelte/lib/Paperclip';
import Pencil from 'phosphor-svelte/lib/Pencil';
import Pepper from 'phosphor-svelte/lib/Pepper';
import Pizza from 'phosphor-svelte/lib/Pizza';
import Plant from 'phosphor-svelte/lib/Plant';
import Popcorn from 'phosphor-svelte/lib/Popcorn';
import PuzzlePiece from 'phosphor-svelte/lib/PuzzlePiece';
import Question from 'phosphor-svelte/lib/Question';
import Rabbit from 'phosphor-svelte/lib/Rabbit';
import Rainbow from 'phosphor-svelte/lib/Rainbow';
import Robot from 'phosphor-svelte/lib/Robot';
import Rocket from 'phosphor-svelte/lib/Rocket';
import Ruler from 'phosphor-svelte/lib/Ruler';
import Sailboat from 'phosphor-svelte/lib/Sailboat';
import Scissors from 'phosphor-svelte/lib/Scissors';
import SealCheck from 'phosphor-svelte/lib/SealCheck';
import ShootingStar from 'phosphor-svelte/lib/ShootingStar';
import Shovel from 'phosphor-svelte/lib/Shovel';
import Smiley from 'phosphor-svelte/lib/Smiley';
import Sneaker from 'phosphor-svelte/lib/Sneaker';
import Snowflake from 'phosphor-svelte/lib/Snowflake';
import SoccerBall from 'phosphor-svelte/lib/SoccerBall';
import Sock from 'phosphor-svelte/lib/Sock';
import Square from 'phosphor-svelte/lib/Square';
import Star from 'phosphor-svelte/lib/Star';
import Sun from 'phosphor-svelte/lib/Sun';
import SunHorizon from 'phosphor-svelte/lib/SunHorizon';
import Target from 'phosphor-svelte/lib/Target';
import Tent from 'phosphor-svelte/lib/Tent';
import Tooth from 'phosphor-svelte/lib/Tooth';
import Train from 'phosphor-svelte/lib/Train';
import Tree from 'phosphor-svelte/lib/Tree';
import Triangle from 'phosphor-svelte/lib/Triangle';
import TShirt from 'phosphor-svelte/lib/TShirt';
import Umbrella from 'phosphor-svelte/lib/Umbrella';

/** Palette for figures; mirrors questions.Colors in Go. Chosen to stay distinct for common colour-vision deficiencies when paired with shape. */
export const palette: Record<string, string> = {
	red: '#e5484d', orange: '#ff8a4c', yellow: '#f5b82e', green: '#35b36f', blue: '#3a8ee8',
	purple: '#7c5ce6', pink: '#e5579a', brown: '#a86b43', gray: '#8a8fa8'
};

/** Colour names in Indonesian, for alt text (never rely on colour alone, but name it when it is the point). */
export const colorName: Record<string, string> = {
	red: 'merah', orange: 'oranye', yellow: 'kuning', green: 'hijau', blue: 'biru', purple: 'ungu', pink: 'merah muda', brown: 'cokelat', gray: 'abu-abu'
};

type FigureMeta = { label: string; color: string };

// Approved question figures: label + default colour come from the shared JSON.
const meta: Record<string, FigureMeta> = Object.fromEntries(
	(data as { name: string; label: string; color: string }[]).map((f) => [f.name, { label: f.label, color: f.color }])
);

// UI-only figures (avatars, badges, rewards) that are not question stimuli.
const uiMeta: Record<string, FigureMeta> = {
	robot: { label: 'robot', color: 'blue' },
	target: { label: 'target', color: 'red' },
	'seal-check': { label: 'centang', color: 'green' },
	fire: { label: 'api', color: 'orange' },
	compass: { label: 'kompas', color: 'blue' },
	'graduation-cap': { label: 'topi wisuda', color: 'purple' },
	flag: { label: 'bendera', color: 'red' },
	'sun-horizon': { label: 'matahari terbit', color: 'orange' },
	'shooting-star': { label: 'bintang jatuh', color: 'purple' }
};

const icons: Record<string, Component<any>> = {
	'orange': Orange, 'cherries': Cherries, 'carrot': Carrot, 'pepper': Pepper, 'avocado': Avocado,
	'cake': Cake, 'cookie': Cookie, 'pizza': Pizza, 'ice-cream': IceCream, 'bread': Bread, 'egg': Egg,
	'hamburger': Hamburger, 'popcorn': Popcorn, 'cheese': Cheese, 'bowl-food': BowlFood, 'fork-knife': ForkKnife,
	'star': Star, 'heart': Heart, 'sun': Sun, 'moon': Moon, 'cloud': Cloud, 'cloud-rain': CloudRain, 'drop': Drop,
	'snowflake': Snowflake, 'lightning': Lightning, 'rainbow': Rainbow, 'flower': Flower, 'flower-tulip': FlowerTulip,
	'leaf': Leaf, 'plant': Plant, 'tree': Tree, 'acorn': Acorn, 'cactus': Cactus,
	'fish': Fish, 'bird': Bird, 'cat': Cat, 'dog': Dog, 'rabbit': Rabbit, 'butterfly': Butterfly, 'cow': Cow, 'horse': Horse, 'bug': Bug,
	'car': Car, 'bus': Bus, 'boat': Boat, 'train': Train, 'airplane': Airplane, 'bicycle': Bicycle, 'rocket': Rocket, 'sailboat': Sailboat,
	'book': Book, 'book-open': BookOpen, 'notebook': Notebook, 'backpack': Backpack, 'pencil': Pencil, 'ruler': Ruler,
	'palette': Palette, 'paint-brush': PaintBrush, 'puzzle': PuzzlePiece, 'balloon': Balloon, 'soccer-ball': SoccerBall,
	'basketball': Basketball, 'gift': Gift, 'bell': Bell, 'house': House, 'tent': Tent, 'bed': Bed, 'tooth': Tooth,
	'hand-soap': HandSoap, 'shovel': Shovel, 't-shirt': TShirt, 'sneaker': Sneaker, 'sock': Sock, 'boot': Boot,
	'umbrella': Umbrella, 'crown': Crown, 'clock': Clock, 'calendar': Calendar, 'globe': Globe, 'map-trifold': MapTrifold,
	'hand': Hand, 'scissors': Scissors, 'eraser': Eraser, 'paperclip': Paperclip, 'chalkboard': Chalkboard, 'smiley': Smiley,
	'circle': Circle, 'triangle': Triangle, 'square': Square, 'hexagon': Hexagon, 'diamond': Diamond, 'question': Question,
	'robot': Robot, 'target': Target, 'seal-check': SealCheck, 'fire': Fire, 'compass': Compass,
	'graduation-cap': GraduationCap, 'flag': Flag, 'sun-horizon': SunHorizon, 'shooting-star': ShootingStar
};

export const figures: Record<string, { icon: Component<any>; color: string; label: string }> = Object.fromEntries(
	Object.entries(icons).map(([name, icon]) => {
		const m = meta[name] ?? uiMeta[name] ?? { label: name, color: 'gray' };
		return [name, { icon, color: m.color, label: m.label }];
	})
);

export function figure(name: string) {
	return figures[name] ?? figures['question'];
}
