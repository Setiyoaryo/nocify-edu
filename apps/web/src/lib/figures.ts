// Every pictured thing in EduPlay (question objects, avatars, badges, rewards) is a
// Phosphor icon (MIT) rendered in its "fill" weight: one consistent, licensed family and
// no emoji (see docs/brand-assets.md). Question names must mirror
// apps/api/internal/questions/figures.go; a Go test checks this file covers them.
import type { Component } from 'svelte';
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
import Bread from 'phosphor-svelte/lib/Bread';
import Bug from 'phosphor-svelte/lib/Bug';
import Bus from 'phosphor-svelte/lib/Bus';
import Butterfly from 'phosphor-svelte/lib/Butterfly';
import Cake from 'phosphor-svelte/lib/Cake';
import Car from 'phosphor-svelte/lib/Car';
import Carrot from 'phosphor-svelte/lib/Carrot';
import Cat from 'phosphor-svelte/lib/Cat';
import Cherries from 'phosphor-svelte/lib/Cherries';
import Circle from 'phosphor-svelte/lib/Circle';
import Cloud from 'phosphor-svelte/lib/Cloud';
import CloudRain from 'phosphor-svelte/lib/CloudRain';
import Compass from 'phosphor-svelte/lib/Compass';
import Cookie from 'phosphor-svelte/lib/Cookie';
import Cow from 'phosphor-svelte/lib/Cow';
import Diamond from 'phosphor-svelte/lib/Diamond';
import Dog from 'phosphor-svelte/lib/Dog';
import Drop from 'phosphor-svelte/lib/Drop';
import Egg from 'phosphor-svelte/lib/Egg';
import Fire from 'phosphor-svelte/lib/Fire';
import Fish from 'phosphor-svelte/lib/Fish';
import Flag from 'phosphor-svelte/lib/Flag';
import Flower from 'phosphor-svelte/lib/Flower';
import FlowerTulip from 'phosphor-svelte/lib/FlowerTulip';
import Gift from 'phosphor-svelte/lib/Gift';
import GraduationCap from 'phosphor-svelte/lib/GraduationCap';
import Hamburger from 'phosphor-svelte/lib/Hamburger';
import HandSoap from 'phosphor-svelte/lib/HandSoap';
import Heart from 'phosphor-svelte/lib/Heart';
import Hexagon from 'phosphor-svelte/lib/Hexagon';
import Horse from 'phosphor-svelte/lib/Horse';
import House from 'phosphor-svelte/lib/House';
import IceCream from 'phosphor-svelte/lib/IceCream';
import Leaf from 'phosphor-svelte/lib/Leaf';
import Lightning from 'phosphor-svelte/lib/Lightning';
import Moon from 'phosphor-svelte/lib/Moon';
import Orange from 'phosphor-svelte/lib/Orange';
import Palette from 'phosphor-svelte/lib/Palette';
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
import SealCheck from 'phosphor-svelte/lib/SealCheck';
import ShootingStar from 'phosphor-svelte/lib/ShootingStar';
import Shovel from 'phosphor-svelte/lib/Shovel';
import Snowflake from 'phosphor-svelte/lib/Snowflake';
import SoccerBall from 'phosphor-svelte/lib/SoccerBall';
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

/** Palette for figures; mirrors questions.Colors in Go. Chosen to stay distinct for common colour-vision deficiencies when paired with shape. */
export const palette: Record<string, string> = {
	red: '#e5484d', orange: '#ff8a4c', yellow: '#f5b82e', green: '#35b36f', blue: '#3a8ee8',
	purple: '#7c5ce6', pink: '#e5579a', brown: '#a86b43', gray: '#8a8fa8'
};

/** Colour names in Indonesian, for alt text (never rely on colour alone, but name it when it is the point). */
export const colorName: Record<string, string> = {
	red: 'merah', orange: 'oranye', yellow: 'kuning', green: 'hijau', blue: 'biru', purple: 'ungu', pink: 'merah muda', brown: 'cokelat', gray: 'abu-abu'
};

export const figures: Record<string, { icon: Component<any>; color: string; label: string }> = {
	'orange': { icon: Orange, color: 'orange', label: 'jeruk' },
	'cherries': { icon: Cherries, color: 'red', label: 'ceri' },
	'carrot': { icon: Carrot, color: 'orange', label: 'wortel' },
	'pepper': { icon: Pepper, color: 'red', label: 'cabai' },
	'avocado': { icon: Avocado, color: 'green', label: 'alpukat' },
	'cake': { icon: Cake, color: 'pink', label: 'kue' },
	'cookie': { icon: Cookie, color: 'brown', label: 'biskuit' },
	'pizza': { icon: Pizza, color: 'orange', label: 'pizza' },
	'ice-cream': { icon: IceCream, color: 'pink', label: 'es krim' },
	'bread': { icon: Bread, color: 'brown', label: 'roti' },
	'egg': { icon: Egg, color: 'yellow', label: 'telur' },
	'hamburger': { icon: Hamburger, color: 'brown', label: 'burger' },
	'popcorn': { icon: Popcorn, color: 'yellow', label: 'popcorn' },
	'star': { icon: Star, color: 'yellow', label: 'bintang' },
	'heart': { icon: Heart, color: 'red', label: 'hati' },
	'sun': { icon: Sun, color: 'yellow', label: 'matahari' },
	'moon': { icon: Moon, color: 'purple', label: 'bulan' },
	'cloud': { icon: Cloud, color: 'blue', label: 'awan' },
	'cloud-rain': { icon: CloudRain, color: 'blue', label: 'awan hujan' },
	'drop': { icon: Drop, color: 'blue', label: 'tetes air' },
	'snowflake': { icon: Snowflake, color: 'blue', label: 'kepingan salju' },
	'lightning': { icon: Lightning, color: 'yellow', label: 'petir' },
	'rainbow': { icon: Rainbow, color: 'purple', label: 'pelangi' },
	'flower': { icon: Flower, color: 'pink', label: 'bunga' },
	'flower-tulip': { icon: FlowerTulip, color: 'red', label: 'bunga tulip' },
	'leaf': { icon: Leaf, color: 'green', label: 'daun' },
	'plant': { icon: Plant, color: 'green', label: 'tanaman' },
	'tree': { icon: Tree, color: 'green', label: 'pohon' },
	'acorn': { icon: Acorn, color: 'brown', label: 'biji' },
	'fish': { icon: Fish, color: 'blue', label: 'ikan' },
	'bird': { icon: Bird, color: 'yellow', label: 'burung' },
	'cat': { icon: Cat, color: 'orange', label: 'kucing' },
	'dog': { icon: Dog, color: 'brown', label: 'anjing' },
	'rabbit': { icon: Rabbit, color: 'pink', label: 'kelinci' },
	'butterfly': { icon: Butterfly, color: 'purple', label: 'kupu-kupu' },
	'cow': { icon: Cow, color: 'brown', label: 'sapi' },
	'horse': { icon: Horse, color: 'brown', label: 'kuda' },
	'bug': { icon: Bug, color: 'red', label: 'serangga' },
	'car': { icon: Car, color: 'blue', label: 'mobil' },
	'bus': { icon: Bus, color: 'yellow', label: 'bus' },
	'boat': { icon: Boat, color: 'blue', label: 'perahu' },
	'train': { icon: Train, color: 'red', label: 'kereta' },
	'airplane': { icon: Airplane, color: 'blue', label: 'pesawat' },
	'bicycle': { icon: Bicycle, color: 'green', label: 'sepeda' },
	'rocket': { icon: Rocket, color: 'red', label: 'roket' },
	'sailboat': { icon: Sailboat, color: 'blue', label: 'perahu layar' },
	'book': { icon: Book, color: 'purple', label: 'buku' },
	'book-open': { icon: BookOpen, color: 'purple', label: 'buku terbuka' },
	'backpack': { icon: Backpack, color: 'red', label: 'tas sekolah' },
	'pencil': { icon: Pencil, color: 'yellow', label: 'pensil' },
	'ruler': { icon: Ruler, color: 'yellow', label: 'penggaris' },
	'palette': { icon: Palette, color: 'purple', label: 'palet' },
	'puzzle': { icon: PuzzlePiece, color: 'green', label: 'puzzle' },
	'balloon': { icon: Balloon, color: 'red', label: 'balon' },
	'soccer-ball': { icon: SoccerBall, color: 'gray', label: 'bola' },
	'basketball': { icon: Basketball, color: 'orange', label: 'bola basket' },
	'gift': { icon: Gift, color: 'pink', label: 'hadiah' },
	'bell': { icon: Bell, color: 'yellow', label: 'lonceng' },
	'house': { icon: House, color: 'orange', label: 'rumah' },
	'tent': { icon: Tent, color: 'green', label: 'tenda' },
	'bed': { icon: Bed, color: 'blue', label: 'tempat tidur' },
	'tooth': { icon: Tooth, color: 'gray', label: 'gigi' },
	'hand-soap': { icon: HandSoap, color: 'blue', label: 'sabun' },
	'shovel': { icon: Shovel, color: 'brown', label: 'sekop' },
	'circle': { icon: Circle, color: 'blue', label: 'lingkaran' },
	'triangle': { icon: Triangle, color: 'blue', label: 'segitiga' },
	'square': { icon: Square, color: 'blue', label: 'persegi' },
	'hexagon': { icon: Hexagon, color: 'blue', label: 'segi enam' },
	'diamond': { icon: Diamond, color: 'blue', label: 'belah ketupat' },
	'question': { icon: Question, color: 'gray', label: 'tanda tanya' },
	'robot': { icon: Robot, color: 'blue', label: 'robot' },
	'target': { icon: Target, color: 'red', label: 'target' },
	'seal-check': { icon: SealCheck, color: 'green', label: 'centang' },
	'fire': { icon: Fire, color: 'orange', label: 'api' },
	'compass': { icon: Compass, color: 'blue', label: 'kompas' },
	'graduation-cap': { icon: GraduationCap, color: 'purple', label: 'topi wisuda' },
	'flag': { icon: Flag, color: 'red', label: 'bendera' },
	'sun-horizon': { icon: SunHorizon, color: 'orange', label: 'matahari terbit' },
	'shooting-star': { icon: ShootingStar, color: 'purple', label: 'bintang jatuh' },
};

export function figure(name: string) {
	return figures[name] ?? figures['question'];
}
