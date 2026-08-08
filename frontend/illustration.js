// Prozedurale Titelillustration: komponiert aus den Eingabefeldern (Ort ->
// Szene, Personen/Tiere -> Figuren, Stimmung -> Himmel/Licht) ein kleines
// SVG-Bilderbuch-Cover - komplett clientseitig, ohne LLM- oder Netzwerk-Calls.
// Der Zufall ist mit dem Geschichten-Titel geseedet, damit dieselbe
// Geschichte immer dasselbe Bild bekommt (auch in der Download-Datei).
(function (global) {
    'use strict';

    const W = 600;
    const H = 170;
    const HORIZON = 96;

    // --- Seeded PRNG (xmur3-Hash + mulberry32) -----------------------------

    function hashSeed(str) {
        let h = 1779033703 ^ str.length;
        for (let i = 0; i < str.length; i++) {
            h = Math.imul(h ^ str.charCodeAt(i), 3432918353);
            h = (h << 13) | (h >>> 19);
        }
        h = Math.imul(h ^ (h >>> 16), 2246822507);
        h = Math.imul(h ^ (h >>> 13), 3266489909);
        return (h ^= h >>> 16) >>> 0;
    }

    function createRng(seedText) {
        let a = hashSeed(seedText);
        return {
            next() {
                a |= 0;
                a = (a + 0x6D2B79F5) | 0;
                let t = Math.imul(a ^ (a >>> 15), 1 | a);
                t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
                return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
            },
            range(min, max) { return min + this.next() * (max - min); },
            int(min, max) { return Math.floor(this.range(min, max + 1)); },
            pick(arr) { return arr[Math.floor(this.next() * arr.length)]; },
            chance(p) { return this.next() < p; }
        };
    }

    // --- SVG-Bausteine ------------------------------------------------------

    function num(v) { return Math.round(v * 10) / 10; }
    function circle(cx, cy, r, fill, extra) {
        return `<circle cx="${num(cx)}" cy="${num(cy)}" r="${num(r)}" fill="${fill}"${extra ? ' ' + extra : ''}/>`;
    }
    function ellipse(cx, cy, rx, ry, fill, extra) {
        return `<ellipse cx="${num(cx)}" cy="${num(cy)}" rx="${num(rx)}" ry="${num(ry)}" fill="${fill}"${extra ? ' ' + extra : ''}/>`;
    }
    function rect(x, y, w, h, fill, rx, extra) {
        return `<rect x="${num(x)}" y="${num(y)}" width="${num(w)}" height="${num(h)}" fill="${fill}"${rx ? ` rx="${num(rx)}"` : ''}${extra ? ' ' + extra : ''}/>`;
    }
    function path(d, fill, extra) {
        return `<path d="${d}" fill="${fill}"${extra ? ' ' + extra : ''}/>`;
    }
    function polygon(points, fill, extra) {
        return `<polygon points="${points}" fill="${fill}"${extra ? ' ' + extra : ''}/>`;
    }
    function line(x1, y1, x2, y2, stroke, width, extra) {
        return `<line x1="${num(x1)}" y1="${num(y1)}" x2="${num(x2)}" y2="${num(y2)}" stroke="${stroke}" stroke-width="${width}" stroke-linecap="round"${extra ? ' ' + extra : ''}/>`;
    }
    function group(transform, content) {
        return `<g transform="${transform}">${content}</g>`;
    }

    // Vierzackiger Funkel-Stern
    function sparkle(cx, cy, r, fill) {
        const d = `M${num(cx)} ${num(cy - r)} Q${num(cx + r * 0.18)} ${num(cy - r * 0.18)} ${num(cx + r)} ${num(cy)} ` +
            `Q${num(cx + r * 0.18)} ${num(cy + r * 0.18)} ${num(cx)} ${num(cy + r)} ` +
            `Q${num(cx - r * 0.18)} ${num(cy + r * 0.18)} ${num(cx - r)} ${num(cy)} ` +
            `Q${num(cx - r * 0.18)} ${num(cy - r * 0.18)} ${num(cx)} ${num(cy - r)} Z`;
        return path(d, fill);
    }

    // --- Wortlisten: Eingabetext -> Szene / Figuren / Stimmung -------------

    // Wörter werden per Präfix verglichen ("Drachen" trifft "drache"); bei
    // mehreren Treffern gewinnt das längste Schlüsselwort ("Prinzessin"
    // schlägt "Prinz").
    const MOOD_KEYWORDS = [
        ['fröhlich', 'tag'], ['froh', 'tag'], ['glücklich', 'tag'], ['gluecklich', 'tag'],
        ['lustig', 'tag'], ['witzig', 'tag'], ['heiter', 'tag'], ['albern', 'tag'],
        ['ausgelassen', 'tag'], ['sonnig', 'tag'],
        ['spannend', 'sonnenuntergang'], ['aufregend', 'sonnenuntergang'],
        ['abenteuer', 'sonnenuntergang'], ['mutig', 'sonnenuntergang'],
        ['wild', 'sonnenuntergang'], ['dramatisch', 'sonnenuntergang'],
        ['geheimnis', 'sonnenuntergang'], ['mysteriös', 'sonnenuntergang'],
        ['rätselhaft', 'sonnenuntergang'],
        ['beruhigend', 'abend'], ['ruhig', 'abend'], ['sanft', 'abend'],
        ['entspann', 'abend'], ['verträumt', 'abend'], ['träumerisch', 'abend'],
        ['müde', 'abend'], ['schläfrig', 'abend'], ['gemütlich', 'abend'],
        ['friedlich', 'abend'], ['einschlaf', 'abend'],
        ['gruselig', 'nacht'], ['unheimlich', 'nacht'], ['dunkel', 'nacht'],
        ['schaurig', 'nacht'], ['gespenstisch', 'nacht'], ['düster', 'nacht'],
        ['traurig', 'regen'], ['melancholisch', 'regen'], ['nachdenklich', 'regen'],
        ['regnerisch', 'regen'], ['regen', 'regen'], ['trüb', 'regen']
    ];

    const SCENE_KEYWORDS = [
        ['wald', 'wald'], ['wäld', 'wald'], ['baum', 'wald'], ['bäume', 'wald'],
        ['dschungel', 'wald'], ['urwald', 'wald'], ['lichtung', 'wald'],
        ['schloss', 'schloss'], ['schlöss', 'schloss'], ['burg', 'schloss'],
        ['königreich', 'schloss'], ['palast', 'schloss'], ['turm', 'schloss'], ['türm', 'schloss'],
        ['meer', 'wasser'], ['see', 'wasser'], ['strand', 'wasser'], ['insel', 'wasser'],
        ['ozean', 'wasser'], ['fluss', 'wasser'], ['flüss', 'wasser'], ['teich', 'wasser'],
        ['bach', 'wasser'], ['hafen', 'wasser'], ['küste', 'wasser'],
        ['unterwasser', 'wasser'], ['wasser', 'wasser'],
        ['berg', 'berge'], ['gebirge', 'berge'], ['alpen', 'berge'], ['gipfel', 'berge'],
        ['hügel', 'berge'], ['huegel', 'berge'], ['felsen', 'berge'],
        ['wiese', 'wiese'], ['garten', 'wiese'], ['gärten', 'wiese'], ['park', 'wiese'],
        ['feld', 'wiese'], ['blumen', 'wiese'], ['bauernhof', 'wiese'],
        ['stadt', 'stadt'], ['städt', 'stadt'], ['dorf', 'stadt'], ['dörf', 'stadt'],
        ['haus', 'stadt'], ['häus', 'stadt'], ['straße', 'stadt'], ['strasse', 'stadt'],
        ['schule', 'stadt'], ['markt', 'stadt'], ['zuhause', 'stadt'], ['daheim', 'stadt'],
        ['weltraum', 'weltraum'], ['weltall', 'weltraum'], ['mond', 'weltraum'],
        ['planet', 'weltraum'], ['rakete', 'weltraum'], ['raumschiff', 'weltraum'],
        ['galaxie', 'weltraum'], ['stern', 'weltraum'], ['mars', 'weltraum'],
        ['höhle', 'hoehle'], ['hoehle', 'hoehle'], ['grotte', 'hoehle'],
        ['bergwerk', 'hoehle'], ['mine', 'hoehle'], ['tunnel', 'hoehle'],
        ['schnee', 'winter'], ['winter', 'winter'], ['eis', 'winter'],
        ['nordpol', 'winter'], ['iglu', 'winter'], ['gletscher', 'winter']
    ];

    const CHARACTER_KEYWORDS = [
        ['hase', 'hase'], ['häschen', 'hase'], ['häsin', 'hase'], ['kaninchen', 'hase'],
        ['fuchs', 'fuchs'], ['füchs', 'fuchs'],
        ['bär', 'baer'], ['baer', 'baer'], ['teddy', 'baer'],
        ['katze', 'katze'], ['kater', 'katze'], ['kätzchen', 'katze'], ['miez', 'katze'],
        ['hund', 'hund'], ['hünd', 'hund'], ['welpe', 'hund'], ['dackel', 'hund'],
        ['maus', 'maus'], ['mäus', 'maus'],
        ['igel', 'igel'],
        ['eule', 'eule'], ['uhu', 'eule'], ['käuzchen', 'eule'],
        ['vogel', 'vogel'], ['vögel', 'vogel'], ['amsel', 'vogel'], ['spatz', 'vogel'],
        ['rabe', 'vogel'], ['papagei', 'vogel'],
        ['frosch', 'frosch'], ['frösch', 'frosch'], ['kröte', 'frosch'],
        ['drache', 'drache'],
        ['einhorn', 'einhorn'], ['einhörn', 'einhorn'],
        ['pferd', 'pferd'], ['pony', 'pferd'], ['esel', 'pferd'], ['fohlen', 'pferd'],
        ['prinzessin', 'prinzessin'], ['königin', 'prinzessin'], ['fürstin', 'prinzessin'],
        ['prinz', 'koenig'], ['könig', 'koenig'], ['kaiser', 'koenig'],
        ['ritter', 'ritter'],
        ['hexe', 'hexe'], ['zauberin', 'hexe'],
        ['zauber', 'zauberer'], ['magier', 'zauberer'],
        ['fee', 'fee'], ['elfe', 'fee'], ['elf', 'fee'],
        ['kind', 'kind'], ['junge', 'kind'], ['jungs', 'kind'], ['mädchen', 'kind'],
        ['maedchen', 'kind'], ['freund', 'kind'], ['geschwister', 'kind'],
        ['bruder', 'kind'], ['brüder', 'kind'], ['schwester', 'kind'],
        ['schüler', 'kind'], ['schülerin', 'kind'],
        ['roboter', 'roboter'], ['robo', 'roboter'],
        ['pirat', 'pirat'],
        ['gespenst', 'gespenst'], ['geist', 'gespenst'], ['spuk', 'gespenst'],
        ['ente', 'ente'], ['küken', 'ente'],
        ['fisch', 'fisch'], ['delfin', 'fisch'], ['wal', 'fisch'], ['hai', 'fisch'],
        ['elefant', 'elefant'],
        ['löwe', 'loewe'], ['loewe', 'loewe'],
        ['schmetterling', 'schmetterling'], ['falter', 'schmetterling'],
        ['schaf', 'schaf'], ['lamm', 'schaf'], ['lämm', 'schaf']
    ];

    function splitWords(text) {
        return String(text || '').toLowerCase().split(/[^a-zäöüß]+/).filter(Boolean);
    }

    // Findet für ein Wort das längste passende Schlüsselwort-Präfix
    function bestMatch(word, keywords) {
        let best = null;
        for (const [keyword, key] of keywords) {
            if (word.startsWith(keyword) && (!best || keyword.length > best.keyword.length)) {
                best = { keyword, key };
            }
        }
        return best;
    }

    function detectFromText(text, keywords, fallback) {
        let best = null;
        for (const word of splitWords(text)) {
            const match = bestMatch(word, keywords);
            if (match && (!best || match.keyword.length > best.keyword.length)) {
                best = match;
            }
        }
        return best ? best.key : fallback;
    }

    function detectCharacters(text, maxCount) {
        const found = [];
        for (const word of splitWords(text)) {
            const match = bestMatch(word, CHARACTER_KEYWORDS);
            if (match && !found.includes(match.key)) {
                found.push(match.key);
            }
        }
        return found.slice(0, maxCount);
    }

    // --- Stimmungs-Paletten -------------------------------------------------

    const MOODS = {
        tag: {
            skyTop: '#B3DFF0', skyBottom: '#EAF6EC',
            groundFar: '#A9CE90', groundNear: '#8FBE74',
            celestial: 'sun', stars: false, rain: false,
            clouds: '#FFFFFF', overlay: null
        },
        sonnenuntergang: {
            skyTop: '#F6B26B', skyBottom: '#FBE3BC',
            groundFar: '#ABBB74', groundNear: '#8FA35F',
            celestial: 'sun-low', stars: false, rain: false,
            clouds: '#FAD3A2', overlay: 'rgba(240,120,60,0.06)'
        },
        abend: {
            skyTop: '#8A8EC8', skyBottom: '#DCC9E6',
            groundFar: '#82A46F', groundNear: '#6C905C',
            celestial: 'moon', stars: true, rain: false,
            clouds: '#C6BEE2', overlay: 'rgba(80,65,140,0.10)'
        },
        nacht: {
            skyTop: '#353A64', skyBottom: '#6C6194',
            groundFar: '#5D7B58', groundNear: '#4C6949',
            celestial: 'moon', stars: true, rain: false,
            clouds: '#5C6094', overlay: 'rgba(25,25,60,0.16)'
        },
        regen: {
            skyTop: '#A9BCC8', skyBottom: '#DCE5EA',
            groundFar: '#94B184', groundNear: '#7C9F6C',
            celestial: null, stars: false, rain: true,
            clouds: '#8CA0AE', overlay: 'rgba(90,110,130,0.08)'
        }
    };

    // --- Himmel und Wetter --------------------------------------------------

    function paintSky(mood) {
        return `<defs><linearGradient id="mi-sky" x1="0" y1="0" x2="0" y2="1">` +
            `<stop offset="0" stop-color="${mood.skyTop}"/>` +
            `<stop offset="1" stop-color="${mood.skyBottom}"/>` +
            `</linearGradient></defs>` +
            rect(0, 0, W, H, 'url(#mi-sky)');
    }

    function paintStars(rng, count) {
        let out = '';
        for (let i = 0; i < count; i++) {
            const x = rng.range(10, W - 10);
            const y = rng.range(8, HORIZON - 25);
            if (rng.chance(0.25)) {
                out += sparkle(x, y, rng.range(2.5, 4), 'rgba(255,246,200,0.9)');
            } else {
                out += circle(x, y, rng.range(0.8, 1.6), 'rgba(255,250,220,0.85)');
            }
        }
        return out;
    }

    function paintCelestial(rng, mood) {
        if (mood.celestial === 'sun') {
            const x = rng.range(440, 530);
            const y = rng.range(26, 44);
            return circle(x, y, 26, 'rgba(255,222,120,0.35)') +
                circle(x, y, 17, '#FFD66B') +
                circle(x - 5, y - 5, 5, 'rgba(255,255,255,0.45)');
        }
        if (mood.celestial === 'sun-low') {
            const x = rng.range(430, 520);
            const y = rng.range(50, 68);
            return circle(x, y, 28, 'rgba(255,190,110,0.35)') +
                circle(x, y, 18, '#FFC168');
        }
        if (mood.celestial === 'moon') {
            const x = rng.range(440, 530);
            const y = rng.range(24, 42);
            const r = 14;
            // Sichel aus zwei Bögen, damit kein "Masken-Kreis" den Himmel verdeckt
            const d = `M${num(x + 3)} ${num(y - r)} A${r} ${r} 0 1 1 ${num(x + 3)} ${num(y + r)} ` +
                `A${num(r * 1.15)} ${num(r * 1.15)} 0 0 0 ${num(x + 3)} ${num(y - r)} Z`;
            return circle(x + 6, y, 20, 'rgba(255,244,200,0.2)') + path(d, '#FBEFC8');
        }
        return '';
    }

    function paintCloud(x, y, scale, fill) {
        return group(`translate(${num(x)} ${num(y)}) scale(${num(scale)})`,
            ellipse(0, 0, 24, 11, fill) +
            circle(-10, -6, 10, fill) +
            circle(4, -9, 12, fill) +
            circle(14, -3, 9, fill));
    }

    function paintClouds(rng, mood, count) {
        let out = '';
        for (let i = 0; i < count; i++) {
            out += paintCloud(rng.range(40, W - 40), rng.range(18, 62),
                rng.range(0.7, 1.15), mood.clouds);
        }
        return out;
    }

    function paintRain(rng, mood) {
        let out = paintCloud(rng.range(90, 200), rng.range(20, 40), 1.2, mood.clouds) +
            paintCloud(rng.range(340, 480), rng.range(18, 36), 1.0, mood.clouds);
        for (let i = 0; i < 26; i++) {
            const x = rng.range(20, W - 20);
            const y = rng.range(40, H - 30);
            out += line(x, y, x - 2, y + 7, 'rgba(130,160,190,0.55)', 1.6);
        }
        return out;
    }

    // --- Hügel und Szenen-Requisiten ---------------------------------------

    function hillPath(rng, baseY, amp) {
        let d = `M0 ${H} L0 ${num(baseY + rng.range(-amp, amp))}`;
        const segs = 4;
        let x = 0;
        for (let i = 0; i < segs; i++) {
            const cx = x + W / segs / 2;
            const cy = baseY + rng.range(-amp, amp);
            x += W / segs;
            d += ` Q ${num(cx)} ${num(cy)} ${num(x)} ${num(baseY + rng.range(-amp, amp))}`;
        }
        return d + ` L${W} ${H} Z`;
    }

    function paintConifer(x, baseY, h, dark) {
        const w = h * 0.55;
        return polygon(`${num(x - w / 2)},${num(baseY)} ${num(x + w / 2)},${num(baseY)} ${num(x)},${num(baseY - h * 0.55)}`, dark ? '#4E7350' : '#5E8A5E') +
            polygon(`${num(x - w * 0.4)},${num(baseY - h * 0.35)} ${num(x + w * 0.4)},${num(baseY - h * 0.35)} ${num(x)},${num(baseY - h)}`, dark ? '#5A8259' : '#6D9B6A') +
            rect(x - 2, baseY, 4, 6, '#7A5B40');
    }

    function paintRoundTree(x, baseY, h) {
        return rect(x - 2.5, baseY - h * 0.4, 5, h * 0.45, '#8A6647') +
            circle(x, baseY - h * 0.62, h * 0.34, '#6FA060') +
            circle(x - h * 0.16, baseY - h * 0.52, h * 0.24, '#7FB06E') +
            circle(x + h * 0.15, baseY - h * 0.7, h * 0.2, '#8CBC78');
    }

    function paintFlower(rng, x, y) {
        const color = rng.pick(['#F1849C', '#F5C64F', '#8FA8E8', '#F09A5E', '#E8A8D8']);
        return line(x, y, x, y - 6, '#5E8A5E', 1.5) +
            circle(x, y - 8, 3, color) +
            circle(x, y - 8, 1.2, '#FFF3D6');
    }

    function paintBush(x, y, scale) {
        return group(`translate(${num(x)} ${num(y)}) scale(${num(scale)})`,
            circle(-8, -5, 8, '#6FA060') + circle(2, -8, 9, '#7FB06E') + circle(10, -4, 7, '#6FA060'));
    }

    function paintMushroom(x, y) {
        return rect(x - 2, y - 6, 4, 6, '#F2E5D0', 1.5) +
            path(`M${num(x - 6)} ${num(y - 5)} Q${num(x)} ${num(y - 14)} ${num(x + 6)} ${num(y - 5)} Z`, '#E06A5A') +
            circle(x - 2, y - 8, 1.2, '#FBEFE0') + circle(x + 2.5, y - 9, 1, '#FBEFE0');
    }

    function paintHouse(rng, x, baseY, scale) {
        const wall = rng.pick(['#F2E3C8', '#EAD3D8', '#D9E3EA', '#F0DDBB']);
        const roof = rng.pick(['#C96F5E', '#A8756B', '#8E7BA8']);
        return group(`translate(${num(x)} ${num(baseY)}) scale(${num(scale)})`,
            rect(-14, -24, 28, 24, wall, 1.5) +
            polygon('-17,-23 17,-23 0,-38', roof) +
            rect(-5, -12, 10, 12, '#7A5B40', 1.5) +
            rect(6, -19, 6, 6, '#FBD98A', 1));
    }

    function paintCastle(x, baseY, scale) {
        const wall = '#E3DCEC';
        const dark = '#CBC0DE';
        const roof = '#8B6BC0';
        return group(`translate(${num(x)} ${num(baseY)}) scale(${num(scale)})`,
            rect(-34, -34, 68, 34, wall, 2) +
            rect(-46, -52, 20, 52, dark, 2) +
            rect(26, -52, 20, 52, dark, 2) +
            polygon('-48,-50 -24,-50 -36,-72', roof) +
            polygon('24,-50 48,-50 36,-72', roof) +
            rect(-8, -60, 16, 60, wall, 2) +
            polygon('-10,-58 10,-58 0,-80', roof) +
            line(0, -80, 0, -90, '#8A6647', 2) +
            polygon('0,-90 12,-86 0,-82', '#E06A5A') +
            path('M-6 0 L-6 -14 Q0 -20 6 -14 L6 0 Z', '#6B5A8C') +
            rect(-40, -44, 7, 9, '#FBD98A', 1.5) +
            rect(33, -44, 7, 9, '#FBD98A', 1.5) +
            rect(-3.5, -48, 7, 9, '#FBD98A', 1.5));
    }

    function paintMountains(rng) {
        let out = polygon(`0,${HORIZON + 10} 90,18 210,${HORIZON + 10}`, '#9BA8C9') +
            polygon(`150,${HORIZON + 10} 300,8 460,${HORIZON + 10}`, '#8493B8') +
            polygon(`380,${HORIZON + 10} 510,26 640,${HORIZON + 10}`, '#9BA8C9');
        out += polygon('66,42 90,18 114,42 102,38 90,46 78,38', '#F4F6FA');
        out += polygon('266,42 300,8 334,42 318,36 300,46 282,36', '#F4F6FA');
        out += polygon('482,52 510,26 538,52 524,47 510,56 496,47', '#F4F6FA');
        return out;
    }

    function paintBoat(rng, x, y) {
        return group(`translate(${num(x)} ${num(y)})`,
            path('M-18 0 Q0 10 18 0 L12 8 Q0 12 -12 8 Z', '#A8764E') +
            line(0, 0, 0, -26, '#7A5B40', 2) +
            polygon('2,-26 2,-4 20,-6', '#F6F1E3') +
            polygon('-2,-24 -2,-6 -14,-8', '#E8DCC4'));
    }

    function paintWaves(rng, yTop, yBottom, color) {
        let out = '';
        for (let i = 0; i < 9; i++) {
            const x = rng.range(15, W - 40);
            const y = rng.range(yTop, yBottom);
            out += path(`M${num(x)} ${num(y)} Q${num(x + 7)} ${num(y - 4)} ${num(x + 14)} ${num(y)} Q${num(x + 21)} ${num(y - 4)} ${num(x + 28)} ${num(y)}`,
                'none', `stroke="${color}" stroke-width="1.8" stroke-linecap="round"`);
        }
        return out;
    }

    function paintRocket(x, y, scale) {
        return group(`translate(${num(x)} ${num(y)}) scale(${num(scale)}) rotate(18)`,
            path('M0 -30 Q13 -12 13 8 L-13 8 Q-13 -12 0 -30 Z', '#F2EFE8') +
            path('M0 -30 Q6 -20 7 -8 L-7 -8 Q-6 -20 0 -30 Z', '#E06A5A') +
            polygon('-13,8 -22,20 -10,14', '#E06A5A') +
            polygon('13,8 22,20 10,14', '#E06A5A') +
            circle(0, -2, 5.5, '#9ED4E8') +
            circle(0, -2, 5.5, 'none', 'stroke="#C8C2B4" stroke-width="1.5"') +
            path('M-6 14 Q0 30 6 14 Z', '#FFC168') +
            path('M-3 14 Q0 24 3 14 Z', '#FFE9A8'));
    }

    function paintPlanet(x, y, r, color, ringColor) {
        let out = circle(x, y, r, color);
        if (ringColor) {
            out += `<ellipse cx="${num(x)}" cy="${num(y)}" rx="${num(r * 1.7)}" ry="${num(r * 0.5)}" fill="none" stroke="${ringColor}" stroke-width="2.5" transform="rotate(-18 ${num(x)} ${num(y)})"/>`;
        }
        return out;
    }

    function paintSnowman(x, baseY) {
        return group(`translate(${num(x)} ${num(baseY)})`,
            circle(0, -10, 11, '#FFFFFF') +
            circle(0, -27, 8, '#FFFFFF') +
            circle(-2.5, -29, 1.2, '#3A3230') + circle(2.5, -29, 1.2, '#3A3230') +
            polygon('0,-26 8,-24.5 0,-23.5', '#F09044') +
            rect(-6, -40, 12, 3, '#3A3230') + rect(-4, -47, 8, 8, '#3A3230', 1) +
            line(-10, -14, -18, -20, '#7A5B40', 2) + line(10, -14, 18, -20, '#7A5B40', 2));
    }

    function paintSnowfall(rng) {
        let out = '';
        for (let i = 0; i < 24; i++) {
            out += circle(rng.range(8, W - 8), rng.range(8, H - 12), rng.range(1, 2.2), 'rgba(255,255,255,0.85)');
        }
        return out;
    }

    // --- Szenen -------------------------------------------------------------
    // Jede Szene liefert Ebenen, die zwischen Himmel und Figuren gestapelt
    // werden. "custom" Szenen (Wasser, Weltraum) malen ihren Boden selbst.

    const SCENES = {
        standard(rng, mood) {
            let front = '';
            for (let i = 0; i < 5; i++) front += paintFlower(rng, rng.range(30, W - 30), rng.range(H - 26, H - 6));
            return {
                back: '',
                mid: paintRoundTree(rng.range(60, 170), 128, 52) + paintBush(rng.range(420, 540), 132, 1),
                front
            };
        },
        wald(rng, mood) {
            let mid = '';
            const xs = [45, 120, 480, 555];
            for (const x of xs) {
                mid += paintConifer(x + rng.range(-15, 15), 128 + rng.range(-4, 4), rng.range(42, 60), rng.chance(0.5));
            }
            mid += paintRoundTree(rng.range(200, 250), 124, 46);
            let front = paintMushroom(rng.range(70, 160), H - 10) + paintBush(rng.range(380, 520), H - 8, 0.9);
            if (rng.chance(0.6)) front += paintMushroom(rng.range(430, 540), H - 8);
            return { back: paintConifer(rng.range(280, 360), HORIZON + 8, 30, true), mid, front };
        },
        schloss(rng, mood) {
            return {
                back: paintCastle(rng.range(280, 340), HORIZON + 16, 0.9),
                mid: paintConifer(rng.range(60, 130), 130, 44, false) + paintRoundTree(rng.range(480, 550), 128, 46),
                front: paintFlower(rng, rng.range(60, 140), H - 12) + paintFlower(rng, rng.range(440, 540), H - 10) + paintFlower(rng, rng.range(180, 260), H - 8)
            };
        },
        berge(rng, mood) {
            return {
                back: paintMountains(rng),
                mid: paintConifer(rng.range(60, 120), 130, 46, false) + paintConifer(rng.range(490, 550), 132, 52, true),
                front: paintFlower(rng, rng.range(150, 250), H - 10) + paintFlower(rng, rng.range(350, 450), H - 8)
            };
        },
        wiese(rng, mood) {
            let front = '';
            for (let i = 0; i < 9; i++) front += paintFlower(rng, rng.range(25, W - 25), rng.range(H - 34, H - 6));
            return {
                back: '',
                mid: paintBush(rng.range(50, 130), 130, 1.1) + paintRoundTree(rng.range(470, 550), 126, 50),
                front
            };
        },
        stadt(rng, mood) {
            let back = '';
            const n = rng.int(3, 4);
            for (let i = 0; i < n; i++) {
                back += paintHouse(rng, 120 + i * rng.range(105, 125) + rng.range(-16, 16), HORIZON + 14, rng.range(0.8, 1.1));
            }
            return {
                back,
                mid: paintRoundTree(rng.range(40, 90), 128, 44),
                front: paintBush(rng.range(440, 550), H - 8, 1) + paintFlower(rng, rng.range(120, 220), H - 10)
            };
        },
        hoehle(rng, mood) {
            // Fels-Rahmen um die Öffnung, Kristalle im Vordergrund
            const rock = '#8C7F76';
            const rockDark = '#77695F';
            const back =
                path(`M0 0 L${W} 0 L${W} ${H} L0 ${H} Z M60 ${H} Q30 60 120 30 Q220 4 300 10 Q400 4 480 30 Q570 62 540 ${H} Z`,
                    rockDark, 'fill-rule="evenodd"') +
                polygon(`150,12 168,52 186,14`, rock) + polygon(`300,8 316,44 332,9`, rock) +
                polygon(`420,16 436,54 452,18`, rock);
            let front = '';
            const crystals = ['#A8E0E8', '#C8B8F0', '#9ED4B8'];
            for (let i = 0; i < 3; i++) {
                const x = rng.pick([rng.range(95, 175), rng.range(420, 500)]);
                const h = rng.range(14, 24);
                front += polygon(`${num(x - 6)},${H} ${num(x + 6)},${H} ${num(x)},${num(H - h)}`, rng.pick(crystals)) +
                    sparkle(x + rng.range(-14, 14), H - h - rng.range(4, 12), 3, 'rgba(255,255,255,0.8)');
            }
            return { back, mid: '', front, groundFar: '#9A8D82', groundNear: '#867970', indoor: true };
        },
        winter(rng, mood) {
            let mid = paintConifer(rng.range(50, 120), 130, 48, true) + paintConifer(rng.range(480, 550), 132, 54, false);
            mid += circle(rng.range(60, 120), 118, 5, '#FFFFFF') + circle(rng.range(480, 545), 120, 5, '#FFFFFF');
            let front = paintSnowfall(rng);
            if (rng.chance(0.7)) front = paintSnowman(rng.range(430, 520), H - 6) + front;
            return {
                back: '',
                mid, front,
                groundFar: '#EDF3F8', groundNear: '#DDE9F2'
            };
        },
        wasser(rng, mood) {
            // Eigener Boden: Wasserband mit Wellen, davor Sandstrand
            const water = mood.stars ? '#5E7FA8' : '#79C0D8';
            const waterDeep = mood.stars ? '#50698F' : '#5FA8C4';
            const sand = mood.stars ? '#C9B98E' : '#EFDCA8';
            let back = rect(0, HORIZON, W, H - HORIZON, water) +
                paintWaves(rng, HORIZON + 6, 126, 'rgba(255,255,255,0.5)') +
                rect(0, HORIZON, W, 4, waterDeep);
            if (rng.chance(0.8)) back += paintBoat(rng, rng.range(90, 220), HORIZON + 16);
            const front = paintBush(rng.range(460, 550), H - 6, 0.8) +
                circle(rng.range(60, 140), H - 10, 3, '#E8C8A0') +
                circle(rng.range(150, 240), H - 6, 2.2, '#D8B890') +
                sparkle(rng.range(340, 420), H - 12, 3, 'rgba(255,255,255,0.7)');
            return {
                back, mid: '', front,
                custom: true,
                sandPath: path(hillPath(rng, 134, 4), sand)
            };
        },
        weltraum(rng, mood) {
            // Übernimmt den kompletten Hintergrund (immer Nacht im All)
            const sky = `<defs><linearGradient id="mi-sky" x1="0" y1="0" x2="0" y2="1">` +
                `<stop offset="0" stop-color="#2B2E55"/><stop offset="1" stop-color="#54487E"/>` +
                `</linearGradient></defs>` + rect(0, 0, W, H, 'url(#mi-sky)');
            let back = paintStars(rng, 26) +
                paintPlanet(rng.range(80, 170), rng.range(26, 50), 13, '#E8A45E', '#F0CE9A') +
                paintPlanet(rng.range(430, 540), rng.range(20, 44), 8, '#9ED4B8', null) +
                paintRocket(rng.range(300, 420), rng.range(30, 60), 0.8);
            // Mondboden mit Kratern
            let craters = '';
            for (let i = 0; i < 4; i++) {
                craters += ellipse(rng.range(50, W - 50), rng.range(140, H - 8), rng.range(7, 13), rng.range(2.5, 4.5), '#9C93B5');
            }
            return {
                back, mid: craters, front: '',
                custom: true,
                overrideSky: sky,
                groundFar: '#BBB2CE', groundNear: '#ABA1C2',
                noWeather: true
            };
        }
    };

    // --- Figuren ------------------------------------------------------------
    // Lokale Koordinaten: Füße bei (0,0), gezeichnet nach oben (negatives y).
    // Fliegende Figuren lassen unten Platz und schweben dadurch über dem Boden.

    const INK = '#3A3230';
    const BLUSH = 'rgba(240,150,140,0.55)';
    const SKIN_TONES = ['#F4CDA6', '#E8B287', '#C68A5B', '#9C6A42'];

    function faceDots(cx, cy, gap) {
        return circle(cx - gap, cy, 1.8, INK) + circle(cx + gap, cy, 1.8, INK) +
            circle(cx - gap - 3.5, cy + 4.5, 2.2, BLUSH) + circle(cx + gap + 3.5, cy + 4.5, 2.2, BLUSH);
    }

    function smile(cx, cy, r) {
        return path(`M${num(cx - r)} ${num(cy)} Q${num(cx)} ${num(cy + r)} ${num(cx + r)} ${num(cy)}`,
            'none', `stroke="${INK}" stroke-width="1.4" stroke-linecap="round"`);
    }

    const CHARACTERS = {
        hase(rng) {
            const fur = '#F5EEE2';
            return ellipse(-7, -55, 5, 15, fur, 'transform="rotate(-10 -7 -55)"') +
                ellipse(7, -55, 5, 15, fur, 'transform="rotate(10 7 -55)"') +
                ellipse(-7, -54, 2.4, 9, '#F4C4C0', 'transform="rotate(-10 -7 -54)"') +
                ellipse(7, -54, 2.4, 9, '#F4C4C0', 'transform="rotate(10 7 -54)"') +
                ellipse(0, -14, 13, 15, fur) +
                ellipse(-6, -2, 5, 3.5, fur) + ellipse(6, -2, 5, 3.5, fur) +
                circle(0, -38, 15, fur) +
                faceDots(0, -39, 5.5) +
                ellipse(0, -34, 2.2, 1.6, '#E8908A') +
                smile(0, -31.5, 2.5);
        },
        fuchs(rng) {
            const fur = '#EF9143';
            return ellipse(15, -10, 9, 5.5, fur, 'transform="rotate(-35 15 -10)"') +
                circle(20, -15, 3.5, '#FBEFE0') +
                ellipse(0, -14, 13, 15, fur) +
                ellipse(0, -9, 8, 10, '#FBE4CB') +
                polygon('-14,-42 -4,-50 -12,-56', fur) + polygon('14,-42 4,-50 12,-56', fur) +
                polygon('-11.5,-45 -6.5,-49 -10.5,-52.5', '#7A4A28') + polygon('11.5,-45 6.5,-49 10.5,-52.5', '#7A4A28') +
                circle(0, -37, 14.5, fur) +
                ellipse(0, -31, 7, 6, '#FBEFE0') +
                faceDots(0, -39, 5.5) +
                circle(0, -33, 2, INK);
        },
        baer(rng) {
            const fur = '#A87D5A';
            return circle(-11, -49, 6, fur) + circle(11, -49, 6, fur) +
                circle(-11, -49, 3, '#C9A379') + circle(11, -49, 3, '#C9A379') +
                ellipse(0, -14, 14, 16, fur) +
                ellipse(0, -9, 8.5, 10, '#C9A379') +
                ellipse(-7, -2, 5.5, 3.5, fur) + ellipse(7, -2, 5.5, 3.5, fur) +
                circle(0, -38, 15, fur) +
                ellipse(0, -32, 7, 5.5, '#E5CDAF') +
                circle(0, -34, 2.4, INK) +
                faceDots(0, -40, 5.5) +
                smile(0, -30.5, 2.5);
        },
        katze(rng) {
            const fur = rng.pick(['#F0A860', '#A8ABBC', '#E8DCC4']);
            return path('M12 -6 Q26 -8 24 -22', 'none', `stroke="${fur}" stroke-width="5" stroke-linecap="round"`) +
                ellipse(0, -13, 12, 14, fur) +
                polygon('-13,-44 -3,-50 -11,-57', fur) + polygon('13,-44 3,-50 11,-57', fur) +
                polygon('-10.5,-46.5 -6,-49.5 -9.5,-53', '#E8908A') + polygon('10.5,-46.5 6,-49.5 9.5,-53', '#E8908A') +
                circle(0, -37, 14, fur) +
                faceDots(0, -38, 5.5) +
                polygon('-1.8,-33.5 1.8,-33.5 0,-31.5', '#E8908A') +
                line(-8, -32, -16, -33.5, INK, 0.9) + line(-8, -30, -16, -29.5, INK, 0.9) +
                line(8, -32, 16, -33.5, INK, 0.9) + line(8, -30, 16, -29.5, INK, 0.9);
        },
        hund(rng) {
            const fur = '#C79A6B';
            return path('M11 -8 Q22 -12 20 -24', 'none', `stroke="${fur}" stroke-width="4.5" stroke-linecap="round"`) +
                ellipse(0, -13, 12.5, 14.5, fur) +
                ellipse(0, -8, 7.5, 9, '#E8D3B4') +
                circle(0, -37, 14, fur) +
                ellipse(-13, -40, 5, 10, '#8F6B45', 'transform="rotate(14 -13 -40)"') +
                ellipse(13, -40, 5, 10, '#8F6B45', 'transform="rotate(-14 13 -40)"') +
                ellipse(0, -31, 7, 5.5, '#E8D3B4') +
                circle(0, -33, 2.4, INK) +
                faceDots(0, -40, 5.5) +
                smile(0, -29.5, 2.5);
        },
        maus(rng) {
            const fur = '#B8B4C4';
            return path('M8 -3 Q22 -2 24 -12', 'none', `stroke="#D8A8B8" stroke-width="2" stroke-linecap="round"`) +
                ellipse(0, -11, 10, 12, fur) +
                circle(-10, -38, 8, fur) + circle(10, -38, 8, fur) +
                circle(-10, -38, 4.5, '#EDC8D2') + circle(10, -38, 4.5, '#EDC8D2') +
                circle(0, -30, 12, fur) +
                faceDots(0, -31, 4.5) +
                circle(0, -26, 1.8, '#E8908A');
        },
        igel(rng) {
            let spikes = '';
            for (let i = 0; i < 7; i++) {
                const a = -160 + i * 23;
                const rad = a * Math.PI / 180;
                const x = 16 * Math.cos(rad), y = -16 + 16 * Math.sin(rad);
                spikes += polygon(`${num(x * 0.5)},${num(-16 + (y + 16) * 0.5)} ${num(x * 1.55)},${num(-16 + (y + 16) * 1.55)} ${num(x * 0.9 + 4)},${num(-16 + (y + 16) * 0.9)}`, '#7A5C43');
            }
            return spikes +
                circle(0, -16, 16, '#93714F') +
                ellipse(2, -12, 12, 11, '#EDD3AE') +
                circle(10, -13, 2.2, INK) +
                circle(3, -15, 1.8, INK) + circle(3, -8, 2.2, BLUSH) +
                ellipse(-6, -1.5, 4, 2.5, '#93714F') + ellipse(7, -1.5, 4, 2.5, '#93714F');
        },
        eule(rng) {
            const fur = '#AE8A64';
            return polygon('-10,-52 -4,-46 -14,-44', fur) + polygon('10,-52 4,-46 14,-44', fur) +
                ellipse(0, -24, 15, 24, fur) +
                ellipse(0, -18, 10, 15, '#E7CFA6') +
                path('M-6 -26 Q0 -21 6 -26', 'none', 'stroke="#C9A379" stroke-width="1.2"') +
                path('M-6 -19 Q0 -14 6 -19', 'none', 'stroke="#C9A379" stroke-width="1.2"') +
                circle(-6.5, -38, 6.5, '#FBF4E4') + circle(6.5, -38, 6.5, '#FBF4E4') +
                circle(-6.5, -38, 2.6, INK) + circle(6.5, -38, 2.6, INK) +
                polygon('-2.5,-33 2.5,-33 0,-28.5', '#F09044') +
                ellipse(-14, -26, 4, 12, '#93714F', 'transform="rotate(12 -14 -26)"') +
                ellipse(14, -26, 4, 12, '#93714F', 'transform="rotate(-12 14 -26)"') +
                line(-4, 0, -4, -3, '#F09044', 2) + line(4, 0, 4, -3, '#F09044', 2);
        },
        vogel(rng) {
            const body = rng.pick(['#6FAEDC', '#E8797F', '#F5C64F']);
            return group('translate(0 -62)',
                ellipse(0, 0, 11, 9.5, body) +
                circle(6, -7, 7, body) +
                polygon('12,-7 19,-5.5 12,-4', '#F09044') +
                circle(8, -8.5, 1.8, INK) +
                ellipse(-4, 0, 6, 4.5, 'rgba(255,255,255,0.4)', 'transform="rotate(-20 -4 0)"') +
                polygon('-10,-2 -18,-6 -11,3', body));
        },
        frosch(rng) {
            const skin = '#86C06C';
            return ellipse(0, -12, 14, 13, skin) +
                ellipse(0, -8, 9, 8, '#D6E8A8') +
                circle(-7, -26, 6, skin) + circle(7, -26, 6, skin) +
                circle(-7, -27, 3, '#FFFFFF') + circle(7, -27, 3, '#FFFFFF') +
                circle(-7, -27, 1.6, INK) + circle(7, -27, 1.6, INK) +
                smile(0, -18, 4) +
                circle(-9, -15, 2.2, BLUSH) + circle(9, -15, 2.2, BLUSH) +
                ellipse(-8, -1.5, 5, 3, skin) + ellipse(8, -1.5, 5, 3, skin);
        },
        drache(rng) {
            const skin = '#7CB86B';
            let spikes = '';
            const spikeXY = [[-4, -55], [4, -57], [11, -53]];
            for (const [sx, sy] of spikeXY) spikes += polygon(`${sx - 3},${sy + 4} ${sx + 3},${sy + 4} ${sx},${sy - 4}`, '#5E9451');
            return path('M-12 -6 Q-26 -2 -28 -14 Q-24 -10 -20 -12', skin,
                'stroke="' + skin + '" stroke-width="4" stroke-linejoin="round"') +
                polygon('-26,-14 -33,-20 -24,-19', '#5E9451') +
                ellipse(0, -16, 13, 16, skin) +
                path('M-10 -26 Q-24 -36 -20 -20 Q-15 -22 -10 -20 Z', '#5E9451') +
                ellipse(0, -11, 8, 10, '#D6E8A8') +
                spikes +
                circle(2, -42, 13, skin) +
                ellipse(11, -37.5, 5, 4, '#A8D096') +
                circle(12.5, -39.5, 1.1, INK) +
                circle(-2, -45, 2, INK) +
                circle(-6, -39, 2.2, BLUSH) +
                ellipse(-6, -1.5, 5, 3.5, skin) + ellipse(6, -1.5, 5, 3.5, skin);
        },
        einhorn(rng) { return horseLike(rng, true); },
        pferd(rng) { return horseLike(rng, false); },
        prinzessin(rng) {
            const skin = rng.pick(SKIN_TONES);
            const dress = rng.pick(['#E88BB0', '#8FA8E8', '#B79ADB']);
            const hair = rng.pick(['#7A4A2A', '#E2B04A', '#3E3228']);
            return path('M0 -36 Q10 -30 14 0 L-14 0 Q-10 -30 0 -36 Z', dress) +
                line(-9, -26, -15, -16, skin, 3.5) + line(9, -26, 15, -16, skin, 3.5) +
                circle(0, -44, 10.5, skin) +
                path('M-10.5 -46 Q-11 -56 0 -56 Q11 -56 10.5 -46 Q10 -51 6 -51 L-6 -51 Q-10 -51 -10.5 -46 Z', hair) +
                path(`M-10.5 -46 Q-13 -36 -9 -28 L-6 -40 Z`, hair) +
                path(`M10.5 -46 Q13 -36 9 -28 L6 -40 Z`, hair) +
                polygon('-6,-54 -3,-59 0,-54 3,-59 6,-54 6,-52 -6,-52', '#F5C64F') +
                faceDots(0, -44, 4) +
                smile(0, -39.5, 2.5);
        },
        koenig(rng) {
            const skin = rng.pick(SKIN_TONES);
            const robe = rng.pick(['#6B8FD4', '#C85A5A', '#7E9E5A']);
            return path('M-8 -32 Q-16 -18 -13 0 L13 0 Q16 -18 8 -32 Z', robe) +
                rect(-3, -30, 6, 30, '#F5C64F', 0, 'opacity="0.5"') +
                line(-10, -26, -16, -15, skin, 3.5) + line(10, -26, 16, -15, skin, 3.5) +
                circle(0, -42, 10.5, skin) +
                path('M-10.5 -44 Q-10 -53 0 -53 Q10 -53 10.5 -44 Q8 -48 0 -48 Q-8 -48 -10.5 -44 Z', '#8A6647') +
                polygon('-7,-52 -4,-58 0,-52 4,-58 7,-52 7,-49 -7,-49', '#F5C64F') +
                circle(0, -57, 1.5, '#E06A5A') +
                faceDots(0, -42, 4) +
                smile(0, -37.5, 2.5);
        },
        ritter(rng) {
            const metal = '#AEB4C0';
            const metalDark = '#8C93A2';
            return path('M-8 -30 Q-13 -16 -11 0 L11 0 Q13 -16 8 -30 Z', metal) +
                rect(-11, -18, 22, 3.5, metalDark) +
                line(-9, -26, -15, -14, metalDark, 3.5) +
                path('M15 -34 L15 -12 Q15 -6 8 -8 Z', '#C85A5A') +
                line(9, -26, 14, -18, metalDark, 3.5) +
                circle(15, -22, 4.5, '#F5C64F') +
                circle(0, -40, 10.5, metal) +
                path('M-10.5 -40 Q-10.5 -51 0 -51 Q10.5 -51 10.5 -40 Z', metalDark) +
                rect(-8, -42, 16, 4, '#2E3644', 2) +
                circle(-4, -40, 1.4, '#9ED4E8') + circle(4, -40, 1.4, '#9ED4E8') +
                path('M0 -51 Q-2 -58 -8 -58 Q-3 -54 -4 -50 Z', '#E06A5A');
        },
        hexe(rng) {
            const skin = rng.pick(SKIN_TONES);
            const dress = '#7E5AA8';
            return line(-16, -2, 18, -34, '#8A6647', 2.5) +
                path('M18 -34 L30 -44 L28 -32 L24 -38 Z', '#C9A35A') +
                path('M0 -34 Q9 -28 13 0 L-13 0 Q-9 -28 0 -34 Z', dress) +
                line(-8, -25, -15, -15, skin, 3.5) + line(8, -25, 14, -19, skin, 3.5) +
                circle(0, -42, 10, skin) +
                path('M-10 -44 Q-12 -32 -8 -27 L-5 -40 Z', '#D97A3F') +
                path('M10 -44 Q12 -32 8 -27 L5 -40 Z', '#D97A3F') +
                ellipse(0, -49, 12, 3, '#4A3A66') +
                polygon('-7,-50 7,-50 2,-64', '#4A3A66') +
                rect(-4, -53, 9, 2.5, '#B79ADB') +
                faceDots(0, -42, 4) + smile(0, -37.5, 2.5);
        },
        zauberer(rng) {
            const robe = '#4A5AA8';
            const skin = rng.pick(SKIN_TONES);
            return line(15, 0, 15, -44, '#8A6647', 2.5) +
                sparkle(15, -48, 5, '#F5C64F') +
                path('M0 -34 Q11 -28 14 0 L-14 0 Q-11 -28 0 -34 Z', robe) +
                sparkle(-6, -18, 2.5, '#F5C64F') + sparkle(5, -8, 2, '#F5C64F') +
                line(-9, -25, -15, -14, robe, 3.5) + line(9, -25, 14, -18, skin, 3) +
                circle(0, -42, 10, skin) +
                path('M-8 -36 Q0 -26 8 -36 Q6 -30 0 -29 Q-6 -30 -8 -36 Z', '#E8E4DC') +
                ellipse(0, -35, 6, 5, '#E8E4DC') +
                ellipse(0, -49, 11.5, 3, '#37418A') +
                polygon('-6.5,-50 6.5,-50 1.5,-65', '#37418A') +
                sparkle(0, -56, 2.2, '#F5C64F') +
                circle(-3.5, -43, 1.6, INK) + circle(3.5, -43, 1.6, INK);
        },
        fee(rng) {
            const skin = rng.pick(SKIN_TONES);
            const dress = rng.pick(['#A8D8A0', '#F0A8C8', '#9ED4E8']);
            return group('translate(0 -26)',
                ellipse(-11, -20, 9, 14, 'rgba(255,255,255,0.65)', 'transform="rotate(20 -11 -20)"') +
                ellipse(11, -20, 9, 14, 'rgba(255,255,255,0.65)', 'transform="rotate(-20 11 -20)"') +
                path('M0 -26 Q7 -21 9 0 L-9 0 Q-7 -21 0 -26 Z', dress) +
                line(-6, -19, -11, -11, skin, 3) +
                line(6, -19, 12, -24, skin, 3) +
                line(12, -24, 16, -30, '#8A6647', 1.5) +
                sparkle(16, -33, 4, '#F5C64F') +
                circle(0, -33, 8.5, skin) +
                path('M-8.5 -35 Q-8 -42 0 -42 Q8 -42 8.5 -35 Q6 -39 0 -39 Q-6 -39 -8.5 -35 Z', '#D97A3F') +
                circle(0, -43, 3, '#D97A3F') +
                faceDots(0, -33, 3.5) + smile(0, -29.5, 2));
        },
        kind(rng) {
            const skin = rng.pick(SKIN_TONES);
            const hair = rng.pick(['#7A4A2A', '#E2B04A', '#3E3228', '#C96F3E']);
            const isDress = rng.chance(0.5);
            let body;
            if (isDress) {
                body = path('M0 -32 Q8 -27 11 -8 L-11 -8 Q-8 -27 0 -32 Z', rng.pick(['#E88BB0', '#F5C64F', '#8FA8E8'])) +
                    line(-3.5, -8, -3.5, 0, skin, 3.5) + line(3.5, -8, 3.5, 0, skin, 3.5);
            } else {
                body = path('M-8 -32 Q-9 -18 -8 -12 L8 -12 Q9 -18 8 -32 Z', rng.pick(['#E8A85A', '#7E9E5A', '#C85A5A'])) +
                    rect(-7.5, -13, 6.5, 13, '#5A6BA8', 2) + rect(1, -13, 6.5, 13, '#5A6BA8', 2);
            }
            const pigtails = isDress && rng.chance(0.6);
            return body +
                line(-8, -27, -13, -17, skin, 3.5) + line(8, -27, 13, -17, skin, 3.5) +
                circle(0, -40, 10, skin) +
                path('M-10 -42 Q-10 -50 0 -50 Q10 -50 10 -42 Q8 -46 0 -46 Q-8 -46 -10 -42 Z', hair) +
                (pigtails ? circle(-11, -44, 3.5, hair) + circle(11, -44, 3.5, hair) : '') +
                faceDots(0, -40, 4) +
                smile(0, -35.5, 2.5);
        },
        roboter(rng) {
            const metal = '#AEB8C4';
            return line(0, -52, 0, -58, '#8C93A2', 2) + circle(0, -60, 2.5, '#E06A5A') +
                rect(-9, -52, 18, 14, '#C6CDD6', 3) +
                rect(-6, -49, 12, 7, '#2E3644', 2) +
                circle(-3, -45.5, 1.6, '#7FE0D8') + circle(3, -45.5, 1.6, '#7FE0D8') +
                rect(-11, -36, 22, 24, metal, 4) +
                rect(-6, -31, 12, 9, '#8C93A2', 2) +
                circle(0, -26.5, 2.5, '#F5C64F') +
                line(-11, -32, -17, -20, '#8C93A2', 3.5) + line(11, -32, 17, -20, '#8C93A2', 3.5) +
                circle(-17, -18, 2.5, '#C6CDD6') + circle(17, -18, 2.5, '#C6CDD6') +
                rect(-8, -12, 6, 12, '#8C93A2', 2) + rect(2, -12, 6, 12, '#8C93A2', 2);
        },
        pirat(rng) {
            const skin = rng.pick(SKIN_TONES);
            return rect(-8, -30, 16, 18, '#FFFFFF', 3) +
                rect(-8, -27, 16, 3, '#C85A5A') + rect(-8, -21, 16, 3, '#C85A5A') +
                rect(-7.5, -13, 6.5, 13, '#3E3228', 2) + rect(1, -13, 6.5, 13, '#3E3228', 2) +
                line(-8, -26, -14, -16, skin, 3.5) + line(8, -26, 14, -16, skin, 3.5) +
                circle(0, -40, 10, skin) +
                path('M-10 -42 Q-10 -51 0 -51 Q10 -51 10 -42 Z', '#C85A5A') +
                circle(8, -49, 2, '#C85A5A') +
                circle(-10, -42, 1.2, '#FFFFFF') + circle(-6, -44.5, 1.2, '#FFFFFF') +
                line(-10, -43.5, 10, -40.5, INK, 1.2) +
                circle(3.5, -41, 3.2, INK) +
                circle(-3.5, -41, 1.6, INK) +
                smile(0, -35.5, 2.5);
        },
        gespenst(rng) {
            return group('translate(0 -24)',
                path('M-13 0 Q-13 -34 0 -34 Q13 -34 13 0 Q9 -5 6.5 0 Q3 -5 0 0 Q-3 -5 -6.5 0 Q-9 -5 -13 0 Z',
                    'rgba(248,246,252,0.95)') +
                circle(-4.5, -20, 2, INK) + circle(4.5, -20, 2, INK) +
                ellipse(0, -14, 2.5, 3.5, INK) +
                circle(-8, -15, 2.2, BLUSH) + circle(8, -15, 2.2, BLUSH));
        },
        ente(rng) {
            const body = '#F5D460';
            return ellipse(0, -10, 12, 10, body) +
                polygon('-9,-14 -19,-10 -9,-7', body) +
                circle(6, -24, 8.5, body) +
                polygon('13,-24 21,-22 13,-19.5', '#F09044') +
                circle(8.5, -25.5, 1.8, INK) +
                circle(3, -21, 2, BLUSH) +
                ellipse(-2, -9, 6, 4.5, '#E8BC4A', 'transform="rotate(-15 -2 -9)"') +
                line(-3, 0, -3, -2, '#F09044', 2) + line(3, 0, 3, -2, '#F09044', 2);
        },
        fisch(rng) {
            const body = rng.pick(['#6FAEDC', '#F0A860', '#E8797F']);
            return group('translate(0 -18)',
                circle(-14, -16, 2, 'rgba(255,255,255,0.6)') + circle(-10, -24, 1.5, 'rgba(255,255,255,0.6)') +
                ellipse(0, 0, 15, 10, body) +
                polygon('-13,0 -24,-8 -24,8', body) +
                polygon('0,-9 6,-16 10,-8', body) +
                circle(7, -2.5, 2, INK) +
                path('M2 2 Q6 5 10 2', 'none', `stroke="${INK}" stroke-width="1.2" stroke-linecap="round"`) +
                path('M-4 -8 Q0 -12 4 -8', 'none', 'stroke="rgba(255,255,255,0.5)" stroke-width="1.4"'));
        },
        elefant(rng) {
            const skin = '#B4B8C4';
            return ellipse(0, -14, 16, 15, skin) +
                ellipse(-14, -36, 8, 10, '#9CA1B0') + ellipse(14, -36, 8, 10, '#9CA1B0') +
                ellipse(-14, -36, 5, 7, '#D3C4CC') + ellipse(14, -36, 5, 7, '#D3C4CC') +
                circle(0, -36, 13, skin) +
                path('M-3 -30 Q-2 -16 6 -12 Q1 -9 -3 -13 Q-6 -21 -6 -30 Z', '#9CA1B0') +
                circle(-5, -38, 1.8, INK) + circle(5, -38, 1.8, INK) +
                circle(-9, -33, 2.2, BLUSH) + circle(9, -33, 2.2, BLUSH) +
                ellipse(-7, -1.5, 5, 3, skin) + ellipse(7, -1.5, 5, 3, skin);
        },
        loewe(rng) {
            const mane = '#D8913F';
            const fur = '#F0C070';
            let maneCircles = '';
            for (let i = 0; i < 8; i++) {
                const a = i * Math.PI / 4;
                maneCircles += circle(Math.cos(a) * 13, -37 + Math.sin(a) * 13, 6.5, mane);
            }
            return ellipse(0, -13, 13, 14, fur) +
                path('M11 -6 Q22 -8 21 -18', 'none', `stroke="${fur}" stroke-width="3" stroke-linecap="round"`) +
                circle(21, -20, 3, mane) +
                maneCircles +
                circle(0, -37, 12.5, fur) +
                ellipse(0, -31.5, 6, 4.5, '#FBEFE0') +
                polygon('-2,-34 2,-34 0,-31.5', '#8F6B45') +
                faceDots(0, -39, 4.5) +
                smile(0, -30, 2);
        },
        schmetterling(rng) {
            const wing = rng.pick(['#E88BB0', '#8FA8E8', '#F5C64F']);
            const wing2 = rng.pick(['#F0C8DC', '#C8D4F0', '#FBE3A8']);
            return group('translate(0 -56)',
                ellipse(-9, -6, 9, 12, wing, 'transform="rotate(24 -9 -6)"') +
                ellipse(9, -6, 9, 12, wing, 'transform="rotate(-24 9 -6)"') +
                ellipse(-8, 7, 7, 8, wing2, 'transform="rotate(-14 -8 7)"') +
                ellipse(8, 7, 7, 8, wing2, 'transform="rotate(14 8 7)"') +
                circle(-9, -7, 3, 'rgba(255,255,255,0.55)') + circle(9, -7, 3, 'rgba(255,255,255,0.55)') +
                ellipse(0, 0, 2.8, 10, '#5A4A3A') +
                circle(0, -9, 3.2, '#5A4A3A') +
                path('M-1.5 -11 Q-4 -16 -6 -16', 'none', 'stroke="#5A4A3A" stroke-width="1.2" stroke-linecap="round"') +
                path('M1.5 -11 Q4 -16 6 -16', 'none', 'stroke="#5A4A3A" stroke-width="1.2" stroke-linecap="round"'));
        },
        schaf(rng) {
            const wool = '#F4F0E6';
            let fluff = '';
            const fluffXY = [[-10, -22], [0, -26], [10, -22], [-12, -13], [12, -13], [-6, -6], [6, -6]];
            for (const [fx, fy] of fluffXY) fluff += circle(fx, fy, 8, wool);
            return fluff +
                ellipse(0, -15, 14, 12, wool) +
                ellipse(0, -32, 8, 9.5, '#8A7A6E') +
                circle(-4, -46, 5, wool) + circle(4, -46, 5, wool) +
                ellipse(-9, -36, 4.5, 2.5, '#6E6058', 'transform="rotate(20 -9 -36)"') +
                ellipse(9, -36, 4.5, 2.5, '#6E6058', 'transform="rotate(-20 9 -36)"') +
                circle(-2.8, -34, 1.5, '#FBF6EC') + circle(2.8, -34, 1.5, '#FBF6EC') +
                circle(-2.8, -34, 0.8, INK) + circle(2.8, -34, 0.8, INK) +
                line(-6, 0, -6, -6, '#8A7A6E', 3) + line(6, 0, 6, -6, '#8A7A6E', 3);
        }
    };

    // Vierbeiner (Einhorn/Pferd) teilen sich eine Zeichnung
    function horseLike(rng, unicorn) {
        const coat = unicorn ? '#F6F1F7' : rng.pick(['#C79A6B', '#8F6B45', '#B4B8C4']);
        const maneColor = unicorn ? rng.pick(['#E8A8C8', '#B79ADB']) : '#5A4A3A';
        let out =
            path('M-22 -22 Q-30 -16 -27 -6', 'none', `stroke="${maneColor}" stroke-width="4" stroke-linecap="round"`) +
            ellipse(0, -22, 21, 11, coat) +
            rect(-16, -18, 5, 18, coat, 2) + rect(-6, -18, 5, 18, coat, 2) +
            rect(3, -18, 5, 18, coat, 2) + rect(12, -18, 5, 18, coat, 2) +
            path('M12 -26 Q16 -42 22 -44 L26 -40 Q28 -34 22 -30 Q17 -28 14 -24 Z', coat) +
            ellipse(25, -41, 5.5, 4.5, coat) +
            path(`M14 -44 Q18 -50 20 -44 Q22 -50 25 -45 L22 -40 Z`, maneColor) +
            path(`M12 -40 Q8 -34 12 -28 L16 -32 Z`, maneColor) +
            circle(24, -42.5, 1.6, INK) +
            circle(28, -39, 1.6, BLUSH) +
            polygon('19,-46 23,-46 21,-51', '#8A7A6E');
        if (unicorn) {
            out += polygon('20,-47 24,-47 22,-58', '#F5C64F') +
                sparkle(28, -52, 3, 'rgba(245,198,79,0.85)');
        }
        return out;
    }

    // Fliegende Figuren stehen nicht auf dem Boden, sondern schweben
    const FLYING = { vogel: true, schmetterling: true, fee: true, gespenst: true, fisch: true };

    // --- Komposition --------------------------------------------------------

    function buildStoryIllustration(params) {
        const titel = String(params.titel || '');
        const thema = String(params.thema || '');
        const ort = String(params.ort || '');
        const personen = String(params.personenTiere || '');
        const stimmung = String(params.stimmung || '');

        const rng = createRng([titel, thema, personen, ort, stimmung].join('|'));

        const moodKey = detectFromText(stimmung, MOOD_KEYWORDS, 'tag');
        const mood = MOODS[moodKey];
        let sceneKey = detectFromText(ort, SCENE_KEYWORDS, null);
        if (!sceneKey) sceneKey = detectFromText(thema, SCENE_KEYWORDS, 'standard');
        const charKeys = detectCharacters(personen, 3);
        if (charKeys.length === 0) charKeys.push('kind');

        const scene = SCENES[sceneKey](rng, mood);

        let svg = `<svg class="story-cover-art" viewBox="0 0 ${W} ${H}" preserveAspectRatio="xMidYMid slice" ` +
            `xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Illustration zur Geschichte">`;

        // Himmel + Wetter
        svg += scene.overrideSky || paintSky(mood);
        if (!scene.noWeather && !scene.indoor) {
            if (mood.stars) svg += paintStars(rng, 14);
            svg += paintCelestial(rng, mood);
            if (!mood.rain) svg += paintClouds(rng, mood, rng.int(2, 3));
        }

        // Hintergrund-Ebene der Szene (Berge, Schloss, Wasser, ...)
        svg += scene.back || '';

        // Hügel (außer bei Szenen mit eigenem Boden)
        const groundFar = scene.groundFar || mood.groundFar;
        const groundNear = scene.groundNear || mood.groundNear;
        if (!scene.custom) {
            svg += path(hillPath(rng, HORIZON + 12, 6), groundFar);
        }
        svg += scene.mid || '';
        if (!scene.custom) {
            svg += path(hillPath(rng, 126, 5), groundNear);
        } else if (scene.sandPath) {
            svg += scene.sandPath;
        } else if (scene.groundNear) {
            svg += path(hillPath(rng, 126, 5), scene.groundNear);
        }

        if (!scene.noWeather && !scene.indoor && mood.rain) svg += paintRain(rng, mood);

        // Stimmungs-Tönung über der Szene, Figuren bleiben davor "im Licht"
        if (mood.overlay && !scene.noWeather) {
            svg += rect(0, 0, W, H, mood.overlay);
        }

        svg += scene.front || '';

        // Figuren mittig auf dem vorderen Hügel verteilen
        const spread = [[300], [262, 340], [222, 302, 382]][charKeys.length - 1];
        for (let i = 0; i < charKeys.length; i++) {
            const key = charKeys[i];
            const x = spread[i] + rng.range(-10, 10);
            const y = FLYING[key] ? rng.range(120, 140) : 158 + rng.range(-4, 4);
            const scale = rng.range(0.92, 1.06);
            const flip = charKeys.length > 1 && i === charKeys.length - 1 && !FLYING[key] && rng.chance(0.6);
            svg += group(`translate(${num(x)} ${num(y)}) scale(${num(flip ? -scale : scale)} ${num(scale)})`,
                CHARACTERS[key](rng));
        }

        // Zauber-Funkeln, wenn das Thema danach klingt
        if (/zauber|magie|magisch|wunder|hexerei|verwunschen/i.test(thema + ' ' + stimmung)) {
            for (let i = 0; i < 5; i++) {
                svg += sparkle(rng.range(120, 480), rng.range(70, 150), rng.range(2.5, 4.5), 'rgba(255,225,130,0.85)');
            }
        }

        svg += '</svg>';
        return svg;
    }

    global.mairchenIllustration = {
        buildStoryIllustration,
        // fürs Testen exportiert
        _internals: { detectFromText, detectCharacters, MOOD_KEYWORDS, SCENE_KEYWORDS, createRng },
        _debugCharacter(key, rng) { return CHARACTERS[key](rng); }
    };
})(typeof window !== 'undefined' ? window : globalThis);
