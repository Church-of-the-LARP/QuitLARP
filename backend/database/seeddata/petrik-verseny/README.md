# Petrik Verseny 2026 – webes feladatrész

Készítsétek el a településetek turisztikai weboldalát a rendelkezésetekre
bocsátott HTML- és CSS-kódok felhasználásával! Mutassátok be városotok vagy
községetek legérdekesebb látnivalóit, népszerű programjait és különlegességeit!
Ha nem ugyanarról a településről érkeztek, akkor annak a településnek a
bemutatását válasszátok, ahol az iskolátok található.

A rendelkezésetekre bocsátott weboldal jelenleg egy képzeletbeli várost,
Kisvárost mutatja be. Ezt kell a feladatleírásnak megfelelően módosítanotok és
kiegészítenetek.

## Feladatok

- Módosítsátok az oldal fejlécét: jelenjen meg rajta a Kisváros név helyett a
  választott település neve és egy hozzá illő szlogen!
- A menüpontok közül csak a legelső (Rólunk) működik jelenleg. Tegyétek
  működőképessé a többi menüpontot! Tanulmányozzátok a navigációs részben és a
  tartalmi részben az első menüpont működését.
- Állítsátok be a navigációs menüt úgy, hogy görgetés közben is az oldal tetején
  maradjon! Ehhez használjátok a Bootstrap megfelelő osztályát: `sticky-top`.
  Az osztályt a navigációs (`nav`) elem `class` paraméterének meglévő értékeihez
  adjátok hozzá.
- Módosítsátok a nyitó részt! Írjatok hozzá egy figyelemfelkeltő címet és egy
  rövid bemutatkozó szöveget!

## A város látnivalói kártyákon helyezkednek el

- A bal oldali kártyán címként jelenítsétek meg a választott település
  valamelyik nevezetességének nevét, a cím alatt pedig egy rövid leírást róla.
- A középső kártyán címként jelenítsétek meg a választott település valamelyik
  parkját, a cím alatt pedig egy rövid leírást róla.
- A jobb oldali kártyán a Belváros cím ne legyen módosítva, csak az alatta lévő
  rövid leírás.
- A weboldalak címsorait a HTML-kódban a `<h1>`, `<h2>`, `<h3>`, `<h4>`,
  `<h5>` és `<h6>` tagekkel adjuk meg. Módosítsátok úgy a kártyákon lévő
  alcímeket, hogy a jelenleginél eggyel alacsonyabb szintűek (és ezáltal
  kisebbek) legyenek.
- A harmadik kártya címe és szövege nincs középre igazítva, az első két kártya
  beállításai alapján igazítsátok azt is középre.

## Programok

- A programokat bemutató részben cseréljétek le a programokat a választott
  település programjaira. Mindhárom program esetében legyen egy cím és egy
  rövid leírás.
- A címek előtt elhelyezkedő emojik az adott program jellegének megfelelőek
  legyenek.
- Módosítsátok úgy a programok alcímeit, hogy a jelenleginél eggyel alacsonyabb
  szintűek (és ezáltal kisebbek) legyenek.
- A Tudtad? rész tartalmazzon egy, a választott településsel kapcsolatos
  érdekességet, különlegességet néhány mondatban bemutatva.
- A láblécben is a választott település neve szerepeljen Kisváros helyett a
  megfelelő helyeken.

## Formázás

- A CSS-ben látható, hogy a weboldalon a betűtípus jelenleg Trebuchet MS,
  módosítsátok egy másik, szintén talp nélküli betűtípusra (sans-serif).
- A jelenlegi oldalon a Bootstrap segítségével van különböző elemek zöld színe
  beállítva. Ezeknek az elemeknek a `class` paramétere olyan osztályneveket
  tartalmaz, amelyekben szerepel a `success` szó, amely a zöld színhez
  szükséges. Ezeknek az elemeknek a színét módosítsátok sárgára (`warning`).

## Fájlok

A kész weboldalt a `deploy/` könyvtárban kell elhelyezni: a belépési pont a
`deploy/index.html`, a stíluslap a `deploy/style.css`.
