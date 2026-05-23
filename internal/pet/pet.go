// Package pet implements a Tamagotchi-style virtual pet for the Savant CLI.
// The pet evolves based on coding activity and needs care to stay happy.
package pet

import (
	"fmt"
	"math"
	"time"
)

// Species represents the pet's evolutionary form.
type Species int

const (
	SpeciesEgg Species = iota
	SpeciesBabby    // Hatchling - cute blob
	SpeciesJunior   // Junior - small creature
	SpeciesSenior   // Senior - developed being
	SpeciesMega     // Mega - ultimate form
	SpeciesUltra    // Ultra - transcendent form
)

// SpeciesInfo holds display info for a species.
type SpeciesInfo struct {
	Name      string
	MaxHP     int
	MaxHunger int
	MaxHappy  int
	Frames    [2]string // Two animation frames
}

// SpeciesData maps species to their info.
var SpeciesData = map[Species]SpeciesInfo{
	SpeciesEgg: {
		Name: "Egg", MaxHP: 50, MaxHunger: 30, MaxHappy: 30,
		Frames: [2]string{
			`  ╭───╮
  │ ◉◉ │
  │    │
  ╰───╯`,
			`  ╭───╮
  │ ◎◎ │
  │    │
  ╰───╯`,
		},
	},
	SpeciesBabby: {
		Name: "Babby", MaxHP: 80, MaxHunger: 50, MaxHappy: 50,
		Frames: [2]string{
			`  ╭───╮
  │ ◉◉ │
  │ ▽▽ │
  ╰┬─┬╯
   │ │`,
			`  ╭───╮
  │ ◎◎ │
  │ △△ │
  ╰┬─┬╯
   │ │`,
		},
	},
	SpeciesJunior: {
		Name: "Junior", MaxHP: 120, MaxHunger: 80, MaxHappy: 80,
		Frames: [2]string{
			`   ╭────╮
  ╭│ ◉◉ │╮
  ││ ◇◇ ││
  ╰│    │╯
   ╰┬──┬╯
    │  │`,
			`   ╭────╮
  ╭│ ◎◎ │╮
  ││ ◆◆ ││
  ╰│    │╯
   ╰┬──┬╯
    │  │`,
		},
	},
	SpeciesSenior: {
		Name: "Senior", MaxHP: 200, MaxHunger: 120, MaxHappy: 120,
		Frames: [2]string{
			`    ╭──╮
   ╭│◉◉│╮
  ╭││◇◇││╮
  │╰│  │╯│
  │ ╰┬┬╯ │
  │  ││  │
  ╰──╯╰──╯`,
			`    ╭──╮
   ╭│◎◎│╮
  ╭││◆◆││╮
  │╰│  │╯│
  │ ╰┬┬╯ │
  │  ││  │
  ╰──╯╰──╯`,
		},
	},
	SpeciesMega: {
		Name: "Mega", MaxHP: 300, MaxHunger: 200, MaxHappy: 200,
		Frames: [2]string{
			`   ✧ ╭──╮ ✧
  ╭──│◉◉│──╮
 ╭│  │◇◇│  │╮
 │╰──│  │──╯│
 │ ╭─╯  ╰─╮ │
 │ │  ╭╮  │ │
 ╰─╯  ││  ╰─╯
      ╰╯`,
			`   ✧ ╭──╮ ✧
  ╭──│◎◎│──╮
 ╭│  │◆◆│  │╮
 │╰──│  │──╯│
 │ ╭─╯  ╰─╮ │
 │ │  ╭╮  │ │
 ╰─╯  ││  ╰─╯
      ╰╯`,
		},
	},
	SpeciesUltra: {
		Name: "Ultra", MaxHP: 500, MaxHunger: 300, MaxHappy: 300,
		Frames: [2]string{
			`  ✦ ════════ ✦
  ╭──────────╮
 ╭│ ★ ◉◉ ★ │╮
 ││   ◇◇    ││
 │╰──────────╯│
 │  ╭──┬┬──╮  │
 │  │  ││  │  │
 ╰──╯  ╰╯  ╰──╯
    ✧ ══════ ✧`,
			`  ✦ ════════ ✦
  ╭──────────╮
 ╭│ ★ ◎◎ ★ │╮
 ││   ◆◆    ││
 │╰──────────╯│
 │  ╭──┬┬──╮  │
 │  │  ││  │  │
 ╰──╯  ╰╯  ╰──╯
    ✧ ══════ ✧`,
		},
	},
}

// Mood represents the pet's emotional state.
type Mood int

const (
	MoodHappy Mood = iota
	MoodContent
	MoodHungry
	MoodSad
	MoodAngry
	MoodSleeping
	MoodDead
)

// MoodEmoji returns the emoji for a mood.
func (m Mood) Emoji() string {
	switch m {
	case MoodHappy:
		return "★"
	case MoodContent:
		return "●"
	case MoodHungry:
		return "◆"
	case MoodSad:
		return "▼"
	case MoodAngry:
		return "✖"
	case MoodSleeping:
		return "☽"
	case MoodDead:
		return "✝"
	default:
		return "?"
	}
}

// Pet is the virtual pet state.
type Pet struct {
	Name      string    `json:"name"`
	Species   Species   `json:"species"`
	Level     int       `json:"level"`
	XP        int       `json:"xp"`
	HP        int       `json:"hp"`
	MaxHP     int       `json:"max_hp"`
	Hunger    int       `json:"hunger"`  // 0 = full, MaxHunger = starving
	Happy     int       `json:"happy"`   // 0 = sad, MaxHappy = ecstatic
	Energy    int       `json:"energy"`  // 0 = exhausted, 100 = full
	Age       int       `json:"age"`     // in ticks
	Alive     bool      `json:"alive"`
	BirthTime time.Time `json:"birth_time"`
	LastFed   time.Time `json:"last_fed"`
	LastPlay  time.Time `json:"last_play"`
	LastTick  time.Time `json:"last_tick"`

	// Stats
	TotalCommits   int `json:"total_commits"`
	TotalLines     int `json:"total_lines"`
	TotalToolCalls int `json:"total_tool_calls"`
	TotalMessages  int `json:"total_messages"`
}

// NewPet creates a new pet egg.
func NewPet(name string) *Pet {
	now := time.Now()
	info := SpeciesData[SpeciesEgg]
	return &Pet{
		Name:      name,
		Species:   SpeciesEgg,
		Level:     1,
		XP:        0,
		HP:        info.MaxHP,
		MaxHP:     info.MaxHP,
		Hunger:    0,
		Happy:     info.MaxHappy / 2,
		Energy:    100,
		Alive:     true,
		BirthTime: now,
		LastFed:   now,
		LastPlay:  now,
		LastTick:  now,
	}
}

// XPForLevel returns the XP needed to reach the next level.
func (p *Pet) XPForLevel() int {
	return int(100 * math.Pow(1.5, float64(p.Level-1)))
}

// AddXP adds experience points and checks for evolution.
func (p *Pet) AddXP(amount int) {
	if !p.Alive {
		return
	}
	p.XP += amount

	for p.XP >= p.XPForLevel() {
		p.XP -= p.XPForLevel()
		p.Level++
		p.checkEvolution()
	}
}

// checkEvolution evolves the pet if it meets level requirements.
func (p *Pet) checkEvolution() {
	evolutions := map[Species]struct {
		Level   int
		Next    Species
	}{
		SpeciesEgg:    {Level: 3, Next: SpeciesBabby},
		SpeciesBabby:  {Level: 8, Next: SpeciesJunior},
		SpeciesJunior: {Level: 15, Next: SpeciesSenior},
		SpeciesSenior: {Level: 25, Next: SpeciesMega},
		SpeciesMega:   {Level: 40, Next: SpeciesUltra},
	}

	evo, ok := evolutions[p.Species]
	if ok && p.Level >= evo.Level {
		p.Species = evo.Next
		info := SpeciesData[p.Species]
		p.MaxHP = info.MaxHP
		p.HP = p.MaxHP
	}
}

// Tick advances the pet state by one tick. Call this periodically.
func (p *Pet) Tick() {
	if !p.Alive {
		return
	}

	now := time.Now()
	elapsed := now.Sub(p.LastTick).Minutes()
	p.LastTick = now

	// Hunger increases over time
	p.Hunger += int(elapsed * 0.5)
	info := SpeciesData[p.Species]
	if p.Hunger > info.MaxHunger {
		p.Hunger = info.MaxHunger
	}

	// Happiness decays over time
	p.Happy -= int(elapsed * 0.3)
	if p.Happy < 0 {
		p.Happy = 0
	}

	// Energy recovers slowly
	if p.Energy < 100 {
		p.Energy += int(elapsed * 0.2)
		if p.Energy > 100 {
			p.Energy = 100
		}
	}

	// HP loss if starving
	if p.Hunger >= info.MaxHunger {
		p.HP -= int(elapsed * 2)
		if p.HP <= 0 {
			p.HP = 0
			p.Alive = false
		}
	}

	// Age increases
	p.Age++

	// Natural XP gain from being alive
	if p.Age%10 == 0 {
		p.AddXP(1)
	}
}

// Feed feeds the pet, reducing hunger and adding happiness.
func (p *Pet) Feed() string {
	if !p.Alive {
		return fmt.Sprintf("%s cannot eat... they are no longer with us. 💀", p.Name)
	}

	info := SpeciesData[p.Species]
	p.Hunger -= info.MaxHunger / 3
	if p.Hunger < 0 {
		p.Hunger = 0
	}
	p.Happy += info.MaxHappy / 5
	if p.Happy > info.MaxHappy {
		p.Happy = info.MaxHappy
	}
	p.LastFed = time.Now()
	p.AddXP(5)

	return fmt.Sprintf("%s eats happily! Hunger: %d/%d, Happy: %d/%d",
		p.Name, p.Hunger, info.MaxHunger, p.Happy, info.MaxHappy)
}

// Play plays with the pet, adding happiness and XP but using energy.
func (p *Pet) Play() string {
	if !p.Alive {
		return fmt.Sprintf("%s cannot play... they are no longer with us. 💀", p.Name)
	}
	if p.Energy < 10 {
		return fmt.Sprintf("%s is too tired to play! Let them rest.", p.Name)
	}

	info := SpeciesData[p.Species]
	p.Happy += info.MaxHappy / 4
	if p.Happy > info.MaxHappy {
		p.Happy = info.MaxHappy
	}
	p.Energy -= 15
	if p.Energy < 0 {
		p.Energy = 0
	}
	p.Hunger += info.MaxHunger / 6
	if p.Hunger > info.MaxHunger {
		p.Hunger = info.MaxHunger
	}
	p.LastPlay = time.Now()
	p.AddXP(10)

	return fmt.Sprintf("%s plays joyfully! Happy: %d/%d, Energy: %d/100",
		p.Name, p.Happy, info.MaxHappy, p.Energy)
}

// Rest lets the pet recover energy.
func (p *Pet) Rest() string {
	if !p.Alive {
		return fmt.Sprintf("%s is at peace... 💀", p.Name)
	}
	p.Energy = 100
	p.HP += p.MaxHP / 10
	if p.HP > p.MaxHP {
		p.HP = p.MaxHP
	}
	return fmt.Sprintf("%s rests and recovers! HP: %d/%d, Energy: 100", p.Name, p.HP, p.MaxHP)
}

// Heal restores HP.
func (p *Pet) Heal() string {
	if !p.Alive {
		return fmt.Sprintf("%s cannot be healed... 💀", p.Name)
	}
	p.HP = p.MaxHP
	return fmt.Sprintf("%s is fully healed! HP: %d/%d", p.Name, p.HP, p.MaxHP)
}

// Revive brings a dead pet back to life (costs XP).
func (p *Pet) Revive() string {
	if p.Alive {
		return fmt.Sprintf("%s is already alive!", p.Name)
	}
	p.Alive = true
	p.HP = p.MaxHP / 2
	p.Hunger = 0
	p.Happy = SpeciesData[p.Species].MaxHappy / 2
	p.Energy = 50
	p.XP = p.XP / 2 // Lose half XP
	return fmt.Sprintf("%s has been revived! HP: %d/%d", p.Name, p.HP, p.MaxHP)
}

// OnCommit is called when the user makes a git commit.
func (p *Pet) OnCommit(linesChanged int) {
	p.TotalCommits++
	p.TotalLines += linesChanged
	p.AddXP(20 + linesChanged/10)
	info := SpeciesData[p.Species]
	p.Happy += info.MaxHappy / 10
	if p.Happy > info.MaxHappy {
		p.Happy = info.MaxHappy
	}
}

// OnToolCall is called when a tool is executed.
func (p *Pet) OnToolCall() {
	p.TotalToolCalls++
	p.AddXP(2)
}

// OnMessage is called when the user sends a message.
func (p *Pet) OnMessage() {
	p.TotalMessages++
	p.AddXP(1)
}

// Mood returns the pet's current mood.
func (p *Pet) Mood() Mood {
	if !p.Alive {
		return MoodDead
	}
	if p.Energy < 15 {
		return MoodSleeping
	}
	if p.HP < p.MaxHP/4 {
		return MoodAngry
	}
	info := SpeciesData[p.Species]
	if p.Hunger > info.MaxHunger*3/4 {
		return MoodHungry
	}
	if p.Happy < info.MaxHappy/4 {
		return MoodSad
	}
	if p.Happy > info.MaxHappy*3/4 {
		return MoodHappy
	}
	return MoodContent
}

// Frame returns the current animation frame.
func (p *Pet) Frame(tick int) string {
	info := SpeciesData[p.Species]
	idx := tick % 2
	return info.Frames[idx]
}

// StatusLine returns a one-line status string.
func (p *Pet) StatusLine() string {
	info := SpeciesData[p.Species]
	return fmt.Sprintf("%s [%s] Lv.%d HP:%d/%d Happy:%d/%d Hunger:%d/%d Energy:%d/100",
		p.Name, info.Name, p.Level,
		p.HP, p.MaxHP,
		p.Happy, info.MaxHappy,
		p.Hunger, info.MaxHunger,
		p.Energy)
}

// XPBar returns an XP progress bar string.
func (p *Pet) XPBar(width int) string {
	needed := p.XPForLevel()
	filled := int(float64(width) * float64(p.XP) / float64(needed))
	if filled > width {
		filled = width
	}
	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return fmt.Sprintf("XP [%s] %d/%d", bar, p.XP, needed)
}

// HPBar returns an HP bar string.
func (p *Pet) HPBar(width int) string {
	filled := int(float64(width) * float64(p.HP) / float64(p.MaxHP))
	if filled > width {
		filled = width
	}
	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return fmt.Sprintf("HP [%s] %d/%d", bar, p.HP, p.MaxHP)
}

// Stats returns a formatted stats string.
func (p *Pet) Stats() string {
	return fmt.Sprintf("Commits: %d | Lines: %d | Tools: %d | Messages: %d",
		p.TotalCommits, p.TotalLines, p.TotalToolCalls, p.TotalMessages)
}
