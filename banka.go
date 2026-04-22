/* Zadatak 3.14 Banka Napisati program koji demonstrira rad banke i izvršavanje zahteve
klijenata za tranfer novca, tj. neku vrstu uplate. Kreirati banku sa 100 klijenata od kojih svaki
mora da ima jedinstven identifikacioni broj računa i početni saldo. Za svakog klijenta kreirati
zasebnu gorutinu koja šalje zahteve banci za tranfer određene količine novca sa računa tog klijenta
na neki drugi račun. Banka obrađuje zahtev klijenta i ukoliko utvrdi da klijent na stanju ima
dovoljnu količinu novca za traženi tranfer, obavlja ga i uslužuje sledećeg klijenta. Prikazivati na
standardnom izlazu tok obrade zahteva klijenata i ukupno stanje u banci nakon svakog izvršenog
transfera.*/

package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type matfWaitGroup struct {
	count     int
	mutex     sync.Mutex
	condition sync.Cond
}

func (self matfWaitGroup) add(n int) {
	self.mutex.Lock()
	self.count += n
	self.mutex.Unlock()
}

func (self matfWaitGroup) done() {
	self.mutex.Lock()
	self.count -= 1

	if self.count == 0 {
		self.condition.Broadcast()
	}
	self.mutex.Unlock()
}

func (self matfWaitGroup) wait() {
	self.mutex.Lock()
	for {
		if self.count == 0 {
			break
		}
		self.condition.Wait()
	}
	self.mutex.Unlock()
}

var wg sync.WaitGroup

type Klijent struct {
	id     int     // identifikacioni broj
	stanje float64 // saldo na racunu
	banka  *Banka  // banka u kojoj je klijent otvorio racun
}

type Banka struct {
	brKlijenata int
	klijenti    []Klijent
	// katanac - sinhronizacija izvrsavanja transfera
	katanac sync.Mutex
	/*
		u slucaju da klijent nema dovoljno novca na racunu
		banka ne izvrsava trazeni transfer
		i stavlja klijenta na cekanje
	*/
	dovoljnoNovca *sync.Cond
}

func napraviKlijente(banka *Banka) {
	// ukoliko je argument referenca na strukturu
	// poljima strukture se pristupa isto tacka notacijom
	banka.dovoljnoNovca = sync.NewCond(&banka.katanac)
	banka.klijenti = make([]Klijent, banka.brKlijenata)
	for i := 0; i < banka.brKlijenata; i++ {
		banka.klijenti[i].id = i
		banka.klijenti[i].stanje = rand.Float64() * 1000.0
		banka.klijenti[i].banka = banka
	}
}

// funkcija koja salje zahteve banci od odredjenog klijenta
func (k Klijent) transfer() {
	for {
		// nasumicno biramo klijenta kome se salje novac i svotu novca
		transferBanka(k.banka, k.id, rand.Intn(k.banka.brKlijenata), rand.Float64()*1000.0)
		// Sleep pauzira tekucu gorutinu
		time.Sleep(time.Millisecond * 100)
	}
}

// funkcija koja obradjuje zahtev klijenta
// id1 - ko salje, id2 - kome se salje
func transferBanka(b *Banka, id1 int, id2 int, kolicina float64) {
	// ulazimo u kriticnu sekciju
	b.dovoljnoNovca.L.Lock()
	// ispisujemo podatke o trazenom transferu
	fmt.Printf("Klijent %d (stanje: %f) zeli da prebaci %f na racun klijenta %d (stanje: %f)\n", id1, b.klijenti[id1].stanje, kolicina, id2, b.klijenti[id2].stanje)
	// sve dok klijent ne ispunjava uslov, stavljamo ga u red cekanja
	for kolicina > b.klijenti[id1].stanje {
		// tekuca gorutina se suspenduje, tj. stavlja u red cekanja
		// red cekanja je FIFO struktura
		// poziv Wait automatski otkljucava kriticnu sekciju
		// zbog toga se Wait sme pozivati samo u okviru kriticne sekcije
		b.dovoljnoNovca.Wait()
	}
	// ako klijent ispunjava uslov, obavljamo transfer
	b.klijenti[id1].stanje -= kolicina
	b.klijenti[id2].stanje += kolicina
	fmt.Printf("Klijent %d (stanje: %f) je prebacio %f na racun klijenta %d (stanje: %f)\n", id1, b.klijenti[id1].stanje, kolicina, id2, b.klijenti[id2].stanje)
	// ispisujemo ukupno stanje u banci
	// ne sme da postoji curenje novca
	b.total()
	// izlazimo iz kriticne sekcije
	b.dovoljnoNovca.L.Unlock()
	/*
		kada je obavljen transfer, signaliziramo svim klijentima
		koji su bili blokirani zbog nedostatka novca
		da ponovo pokusaju sa svojim zahtevom
		jer se mozda nekom od njih u prethodnom transferu uplatio novac
	*/
	b.dovoljnoNovca.Broadcast()
	// funkcija Broadcast signalizira svima iz reda cekanja
	// funkcija Signal signalizira prvoj ubacenoj gorutini u red cekanja
	// tj. onoj koja najduze ceka u redu
}

// metod koji ispisuje ukupno stanje u banci
// provera da li dolazi do "curenja" novca
// odnosno, da li se ispravno obavlja transfer
func (banka Banka) total() {
	stanje := 0.0
	for _, v := range banka.klijenti {
		stanje += v.stanje
	}
	fmt.Printf("Total: %f\n", stanje)

}

func main() {
	banka := Banka{brKlijenata: 100}
	napraviKlijente(&banka)
	fmt.Println(banka)
	banka.total()
	// WaitGroup koristimo samo da ne bi glavna gorutina
	// zavrsila main funkciju i prekinula program
	// dok su gorutine za klijente i dalje aktivne
	wg.Add(banka.brKlijenata)
	for i := 0; i < banka.brKlijenata; i++ {
		go banka.klijenti[i].transfer()
	}
	wg.Wait()
}