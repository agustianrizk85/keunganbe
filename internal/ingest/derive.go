package ingest

import (
	"fmt"

	"greenpark/finance/internal/domain"
)

// deriveAll fills the rule-based war-room sections (alerts, AI insights,
// decisions, KPI scorecard, early-warning triggers) from the assembled figures.
func deriveAll(d *domain.Dashboard) {
	d.Alerts = deriveAlerts(d)
	d.AI = deriveAI(d)
	d.Decisions = deriveDecisions(d)
	d.KPIs = deriveKPIs(d)
	d.Triggers = deriveTriggers(d)
}

func deriveAlerts(d *domain.Dashboard) []domain.Alert {
	s := d.Summary
	var out []domain.Alert
	if s.CancelRate >= 20 {
		out = append(out, domain.Alert{
			Tone: "red", Title: "Rasio batal tinggi",
			Detail: fmt.Sprintf("%d%% transaksi batal (%d deal) — kualitas booking & verifikasi DP perlu diperketat.", s.CancelRate, s.BatalCount),
			Action: "Audit penyebab batal + perketat syarat DP sebelum booking",
		})
	}
	// Projects with a weak akad/batal profile.
	for _, p := range d.Projects {
		if p.Status == domain.StatusRed {
			out = append(out, domain.Alert{
				Tone: "red", Title: "Proyek " + p.Name + " kritis",
				Detail: fmt.Sprintf("%d akad vs %d batal — konversi booking lemah.", p.Akad, p.Batal),
				Action: "Tinjau pricing & proses KPR proyek ini",
			})
		}
	}
	// Stuck pipeline with explicit problems.
	stuck := 0
	for _, r := range d.Pipeline {
		if r.Kendala != "" {
			stuck++
		}
	}
	if stuck > 0 {
		out = append(out, domain.Alert{
			Tone: "orange", Title: "Kendala proses KPR",
			Detail: fmt.Sprintf("%d booking tertahan dengan kendala bank/berkas.", stuck),
			Action: "Kejar bank & lengkapi berkas untuk cairkan SP3",
		})
	}
	if s.KprShare >= 70 {
		out = append(out, domain.Alert{
			Tone: "yellow", Title: "Ketergantungan KPR tinggi",
			Detail: fmt.Sprintf("%d%% akad lewat KPR — arus kas bergantung pada kecepatan pencairan bank.", s.KprShare),
			Action: "Jaga relasi multi-bank & percepat pemberkasan",
		})
	}
	if s.TargetAkad > 0 && s.Achievement < 80 {
		out = append(out, domain.Alert{
			Tone: "orange", Title: "Capaian akad di bawah target",
			Detail: fmt.Sprintf("%d dari target %d akad (%d%%).", s.AkadCount, s.TargetAkad, s.Achievement),
			Action: "Dorong konversi booking aktif jadi akad",
		})
	}
	if len(out) == 0 {
		out = append(out, domain.Alert{
			Tone: "green", Title: "Tidak ada alarm kritis",
			Detail: "Rasio batal terkendali dan pipeline berjalan sesuai jalur.",
			Action: "Pertahankan ritme penagihan DP & pemberkasan KPR",
		})
	}
	return out
}

func deriveAI(d *domain.Dashboard) []domain.AIInsight {
	s := d.Summary
	var out []domain.AIInsight
	out = append(out, domain.AIInsight{
		Type: "Ringkasan Akad", Tone: "navy", Icon: "trend",
		Text: fmt.Sprintf("%d akad senilai Rp %s, cash-in DP Rp %s. %d booking aktif (estimasi Rp %s) menunggu pencairan.",
			s.AkadCount, jutaStr(s.NilaiAkad), jutaStr(s.CashIn), s.BookingCount, jutaStr(s.PipelineValue)),
	})
	if len(d.Banks) > 0 {
		b := d.Banks[0]
		out = append(out, domain.AIInsight{
			Type: "Konsentrasi Bank", Tone: "yellow", Icon: "bank",
			Text: fmt.Sprintf("%s memegang %d%% plafond KPR (Rp %s) — konsentrasi pada satu bank menambah risiko antrian pencairan.", b.Name, b.Share, jutaStr(b.Plafon)),
		})
	}
	if len(d.Projects) > 0 {
		p := d.Projects[0]
		out = append(out, domain.AIInsight{
			Type: "Proyek Penyumbang", Tone: "green", Icon: "rec",
			Text: fmt.Sprintf("%s menyumbang nilai akad terbesar (Rp %s dari %d akad).", p.Name, jutaStr(p.Nilai), p.Akad),
		})
	}
	if s.CancelRate >= 15 {
		out = append(out, domain.AIInsight{
			Type: "Risiko Pembatalan", Tone: "red", Icon: "alert",
			Text: fmt.Sprintf("Rasio batal %d%% — tiap pembatalan menahan unit & menunda cash-in. Fokus kualifikasi calon konsumen.", s.CancelRate),
		})
	}
	if s.AvgDurasi > 0 {
		tone := "green"
		if s.AvgDurasi > 60 {
			tone = "orange"
		}
		out = append(out, domain.AIInsight{
			Type: "Kecepatan Proses", Tone: tone, Icon: "cash",
			Text: fmt.Sprintf("Rata-rata booking→akad %d hari. Pangkas durasi pemberkasan untuk percepat arus kas.", s.AvgDurasi),
		})
	}
	return out
}

func deriveDecisions(d *domain.Dashboard) []domain.Decision {
	s := d.Summary
	out := []domain.Decision{
		{Role: "CEO", Tone: "navy", Text: fmt.Sprintf("Pantau capaian akad %d%% vs target %d", s.Achievement, s.TargetAkad)},
		{Role: "Dirkeu", Tone: "orange", Text: "Jaga cash-in DP & percepat pencairan KPR pipeline"},
	}
	if s.CancelRate >= 15 {
		out = append(out, domain.Decision{Role: "Sales", Tone: "red", Text: fmt.Sprintf("Tekan rasio batal (%d%%) — perketat verifikasi DP", s.CancelRate)})
	}
	stuck := 0
	for _, r := range d.Pipeline {
		if r.Kendala != "" {
			stuck++
		}
	}
	if stuck > 0 {
		out = append(out, domain.Decision{Role: "Collection", Tone: "orange", Text: fmt.Sprintf("Selesaikan %d kendala berkas KPR ke bank", stuck)})
	}
	if len(d.Banks) > 0 && d.Banks[0].Share >= 50 {
		out = append(out, domain.Decision{Role: "Treasury", Tone: "yellow", Text: "Diversifikasi bank KPR — kurangi konsentrasi pencairan"})
	}
	return out
}

func deriveKPIs(d *domain.Dashboard) []domain.KPI {
	s := d.Summary
	state := func(v, green, yellow int, higherBetter bool) string {
		if higherBetter {
			switch {
			case v >= green:
				return "green"
			case v >= yellow:
				return "yellow"
			default:
				return "red"
			}
		}
		switch {
		case v <= green:
			return "green"
		case v <= yellow:
			return "yellow"
		default:
			return "red"
		}
	}
	return []domain.KPI{
		{No: 1, KPI: "Capaian Akad", Def: "Akad vs target tahun ini", PIC: "Dirkeu", Green: "≥95%", Yellow: "80–94%", Red: "<80%", Val: fmt.Sprintf("%d%%", s.Achievement), State: state(s.Achievement, 95, 80, true)},
		{No: 2, KPI: "Cash-in DP", Def: "DP terkumpul (Rp juta)", PIC: "Collection", Green: "naik", Yellow: "stabil", Red: "turun", Val: "Rp " + jutaStr(s.CashIn), State: "green"},
		{No: 3, KPI: "Rasio Batal", Def: "Batal / total transaksi", PIC: "Sales", Green: "≤10%", Yellow: "11–20%", Red: ">20%", Val: fmt.Sprintf("%d%%", s.CancelRate), State: state(s.CancelRate, 10, 20, false)},
		{No: 4, KPI: "KPR Share", Def: "Akad lewat KPR", PIC: "Dirkeu", Green: "≤60%", Yellow: "61–80%", Red: ">80%", Val: fmt.Sprintf("%d%%", s.KprShare), State: state(s.KprShare, 60, 80, false)},
		{No: 5, KPI: "Durasi Booking→Akad", Def: "Rata-rata hari", PIC: "Legal/KPR", Green: "≤45", Yellow: "46–60", Red: ">60", Val: fmt.Sprintf("%d hari", s.AvgDurasi), State: state(s.AvgDurasi, 45, 60, false)},
		{No: 6, KPI: "Booking Aktif", Def: "Pipeline belum akad", PIC: "Sales", Green: "—", Yellow: "—", Red: "—", Val: fmt.Sprintf("%d deal", s.BookingCount), State: "yellow"},
		{No: 7, KPI: "Konsentrasi Bank", Def: "Share bank KPR terbesar", PIC: "Treasury", Green: "≤40%", Yellow: "41–60%", Red: ">60%", Val: bankShareStr(d), State: bankShareState(d, state)},
	}
}

func deriveTriggers(d *domain.Dashboard) []domain.Trigger {
	s := d.Summary
	red := func(cond bool) string {
		if cond {
			return "red"
		}
		return "green"
	}
	return []domain.Trigger{
		{Cond: "Rasio batal tinggi", Thr: ">20%", Status: red(s.CancelRate > 20), PIC: "Sales", Act: "Audit batal + perketat DP", Esc: "Dirkeu"},
		{Cond: "Capaian akad rendah", Thr: "<80%", Status: red(s.TargetAkad > 0 && s.Achievement < 80), PIC: "Dirkeu", Act: "Dorong konversi booking", Esc: "CEO"},
		{Cond: "Durasi proses lambat", Thr: ">60 hari", Status: red(s.AvgDurasi > 60), PIC: "Legal/KPR", Act: "Percepat pemberkasan ke bank", Esc: "Dirkeu"},
		{Cond: "Ketergantungan KPR", Thr: ">80%", Status: red(s.KprShare > 80), PIC: "Treasury", Act: "Dorong skema cash & multi-bank", Esc: "Dirkeu"},
		{Cond: "Kendala berkas KPR", Thr: ">0", Status: red(pipelineStuck(d) > 0), PIC: "Collection", Act: "Kejar bank & lengkapi dokumen", Esc: "Dirkeu"},
	}
}

/* -------------------------------- helpers --------------------------------- */

func pipelineStuck(d *domain.Dashboard) int {
	n := 0
	for _, r := range d.Pipeline {
		if r.Kendala != "" {
			n++
		}
	}
	return n
}

func bankShareStr(d *domain.Dashboard) string {
	if len(d.Banks) == 0 {
		return "—"
	}
	return fmt.Sprintf("%d%%", d.Banks[0].Share)
}

func bankShareState(d *domain.Dashboard, state func(v, g, y int, hb bool) string) string {
	if len(d.Banks) == 0 {
		return "green"
	}
	return state(d.Banks[0].Share, 40, 60, false)
}

// jutaStr renders a Rp-juta figure compactly: ≥1000 juta → "x,y M" (miliar),
// else "n jt".
func jutaStr(juta float64) string {
	if juta >= 1000 {
		return fmt.Sprintf("%.1f M", juta/1000)
	}
	return fmt.Sprintf("%.0f jt", juta)
}
