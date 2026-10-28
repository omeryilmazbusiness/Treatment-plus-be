package domain

import "strings"

// Stat is a Wise highlight cell (15+, 5 branches, …).
type Stat struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func (s Stat) Normalize() Stat {
	return Stat{Value: strings.TrimSpace(s.Value), Label: strings.TrimSpace(s.Label)}
}

type Doctor struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Bio      string `json:"bio"`
	PhotoKey string `json:"photo_key"`
	Stats    []Stat `json:"stats,omitempty"`
}

func (d Doctor) Normalize() Doctor {
	out := Doctor{
		Name:     strings.TrimSpace(d.Name),
		Title:    strings.TrimSpace(d.Title),
		Subtitle: strings.TrimSpace(d.Subtitle),
		Bio:      strings.TrimSpace(d.Bio),
		PhotoKey: strings.TrimSpace(d.PhotoKey),
	}
	for _, st := range d.Stats {
		if n := st.Normalize(); n.Value != "" || n.Label != "" {
			out.Stats = append(out.Stats, n)
		}
	}
	return out
}

type Lab struct {
	Title    string `json:"title"`
	Text     string `json:"text"`
	ImageKey string `json:"image_key"`
	Stats    []Stat `json:"stats,omitempty"`
}

func (l Lab) Normalize() Lab {
	out := Lab{
		Title:    strings.TrimSpace(l.Title),
		Text:     strings.TrimSpace(l.Text),
		ImageKey: strings.TrimSpace(l.ImageKey),
	}
	for _, st := range l.Stats {
		if n := st.Normalize(); n.Value != "" || n.Label != "" {
			out.Stats = append(out.Stats, n)
		}
	}
	return out
}

type Awards struct {
	Text      string   `json:"text"`
	BadgeKeys []string `json:"badge_keys,omitempty"`
}

func (a Awards) Normalize() Awards {
	out := Awards{Text: strings.TrimSpace(a.Text)}
	for _, k := range a.BadgeKeys {
		if t := strings.TrimSpace(k); t != "" {
			out.BadgeKeys = append(out.BadgeKeys, t)
		}
	}
	return out
}

type Hotel struct {
	Name     string `json:"name"`
	Text     string `json:"text"`
	ImageKey string `json:"image_key"`
}

func (h Hotel) Normalize() Hotel {
	return Hotel{
		Name:     strings.TrimSpace(h.Name),
		Text:     strings.TrimSpace(h.Text),
		ImageKey: strings.TrimSpace(h.ImageKey),
	}
}

type Review struct {
	Author   string `json:"author"`
	Rating   int    `json:"rating"`
	Text     string `json:"text"`
	Source   string `json:"source"`
	Date     string `json:"date,omitempty"`
	Verified bool   `json:"verified"`
}

func (r Review) Normalize() Review {
	rating := r.Rating
	if rating < 1 {
		rating = 5
	} else if rating > 5 {
		rating = 5
	}
	src := strings.TrimSpace(r.Source)
	if src == "" {
		src = "Google"
	}
	return Review{
		Author:   strings.TrimSpace(r.Author),
		Rating:   rating,
		Text:     strings.TrimSpace(r.Text),
		Source:   src,
		Date:     strings.TrimSpace(r.Date),
		Verified: r.Verified,
	}
}

type BeforeAfterCase struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Tag            string `json:"tag"`
	Description    string `json:"description"`
	BeforeImageKey string `json:"before_image_key"`
	AfterImageKey  string `json:"after_image_key"`
}

func (b BeforeAfterCase) Normalize() BeforeAfterCase {
	return BeforeAfterCase{
		ID:             strings.TrimSpace(b.ID),
		Title:          strings.TrimSpace(b.Title),
		Tag:            strings.TrimSpace(b.Tag),
		Description:    strings.TrimSpace(b.Description),
		BeforeImageKey: strings.TrimSpace(b.BeforeImageKey),
		AfterImageKey:  strings.TrimSpace(b.AfterImageKey),
	}
}

type CTAConfig struct {
	WhatsAppNumber   string `json:"whatsapp_number"`
	WhatsAppMessage  string `json:"whatsapp_message"`
	PhoneNumber      string `json:"phone_number"`
	BookingURL       string `json:"booking_url"`
	EnableAcceptance bool   `json:"enable_acceptance"`
}

func (c CTAConfig) Normalize() CTAConfig {
	return CTAConfig{
		WhatsAppNumber:   strings.TrimSpace(c.WhatsAppNumber),
		WhatsAppMessage:  strings.TrimSpace(c.WhatsAppMessage),
		PhoneNumber:      strings.TrimSpace(c.PhoneNumber),
		BookingURL:       strings.TrimSpace(c.BookingURL),
		EnableAcceptance: c.EnableAcceptance,
	}
}
