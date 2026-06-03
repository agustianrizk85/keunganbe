package repository

import (
	"greenpark/finance/internal/domain"
	"greenpark/finance/internal/passwd"
)

// This file holds the representative seed data for the dashboard. It mirrors the
// figures shown to the CEO war-room and is intended to be replaced by a real
// data source (accounting / ERP) behind the same FinanceRepository interface.
//
// Monetary values are in millions of Rupiah (Rp juta).

func seedProjects() []domain.Project {
	return []domain.Project{
		{ID: "premiere", Name: "The Hauz Premiere", Units: 30, Budget: 19500, Spent: 6400, Revenue: 27000, Collected: 9200, Margin: 28, Status: domain.StatusGreen, PIC: "Finance — Rina", CashNote: "Arus kas sehat, DP lancar", Decision: "—"},
		{ID: "garden", Name: "The Hauz Garden", Units: 28, Budget: 18200, Spent: 4100, Revenue: 24600, Collected: 6400, Margin: 26, Status: domain.StatusGreen, PIC: "Finance — Rina", CashNote: "Tahap awal, sesuai rencana", Decision: "—"},
		{ID: "aurora", Name: "Greenpark Aurora", Units: 44, Budget: 29800, Spent: 24600, Revenue: 41000, Collected: 33500, Margin: 27, Status: domain.StatusGreen, PIC: "Finance — Doni", CashNote: "Margin terjaga, KPR cair lancar", Decision: "—"},
		{ID: "limo", Name: "Z Hauz Limo", Units: 42, Budget: 28400, Spent: 20800, Revenue: 38600, Collected: 27200, Margin: 24, Status: domain.StatusYellow, PIC: "Finance — Doni", CashNote: "Biaya material naik — margin menipis", Decision: "Review harga jual sisa unit"},
		{ID: "serpong", Name: "Le Hauz Serpong", Units: 36, Budget: 24600, Spent: 19200, Revenue: 32400, Collected: 20100, Margin: 19, Status: domain.StatusYellow, PIC: "Finance — Sari", CashNote: "Penagihan termin lambat", Decision: "Percepat tagih progress"},
		{ID: "cibubur", Name: "Le Hauz Cibubur", Units: 38, Budget: 26800, Spent: 24900, Revenue: 34200, Collected: 17600, Margin: 12, Status: domain.StatusRed, PIC: "Finance — Sari", CashNote: "Cost overrun + koleksi tertahan", Decision: "Eskalasi: audit biaya & tagih DP"},
	}
}

func seedReceivableType() []domain.MetaItem {
	return []domain.MetaItem{
		{Key: "kpr", Label: "KPR Bank", Tone: "green", Note: "Pencairan melalui bank — risiko rendah jika berkas lengkap"},
		{Key: "cash", Label: "Cash Bertahap", Tone: "yellow", Note: "Cicilan langsung konsumen — perlu disiplin penagihan"},
		{Key: "dp", Label: "DP / Booking", Tone: "orange", Note: "Uang muka belum lunas — tahan akad jika menunggak"},
	}
}

func seedAgingMeta() []domain.MetaItem {
	return []domain.MetaItem{
		{Key: "current", Label: "Lancar", Tone: "green", SLA: "Belum jatuh tempo"},
		{Key: "d30", Label: "1–30 Hari", Tone: "yellow", SLA: "Reminder + konfirmasi jadwal bayar"},
		{Key: "d60", Label: "31–60 Hari", Tone: "orange", SLA: "Surat penagihan + kunjungan"},
		{Key: "d90", Label: ">90 Hari", Tone: "red", SLA: "Eskalasi legal / restrukturisasi"},
	}
}

func seedReceivables() []domain.Receivable {
	return []domain.Receivable{
		{ID: "AR-241", Project: "Le Hauz Cibubur", Customer: "Bpk. Hartono", Type: "dp", Amount: 420, Aging: 96, Bucket: "d90", SLA: "overdue", Owner: "Sari", Next: "Eskalasi legal — surat somasi 1"},
		{ID: "AR-238", Project: "Le Hauz Serpong", Customer: "Ibu Wulandari", Type: "cash", Amount: 310, Aging: 64, Bucket: "d60", SLA: "overdue", Owner: "Sari", Next: "Kunjungan penagihan langsung"},
		{ID: "AR-233", Project: "Le Hauz Cibubur", Customer: "Bpk. Saputra", Type: "kpr", Amount: 685, Aging: 41, Bucket: "d60", SLA: "due", Owner: "Doni", Next: "Lengkapi berkas KPR ke bank"},
		{ID: "AR-229", Project: "Z Hauz Limo", Customer: "Ibu Permata", Type: "cash", Amount: 180, Aging: 22, Bucket: "d30", SLA: "due", Owner: "Doni", Next: "Reminder + konfirmasi transfer"},
		{ID: "AR-226", Project: "Greenpark Aurora", Customer: "Bpk. Nugroho", Type: "kpr", Amount: 740, Aging: 14, Bucket: "d30", SLA: "ok", Owner: "Doni", Next: "Tunggu jadwal akad bank"},
		{ID: "AR-221", Project: "The Hauz Premiere", Customer: "Ibu Halim", Type: "kpr", Amount: 560, Aging: 0, Bucket: "current", SLA: "ok", Owner: "Rina", Next: "On schedule — akad H+9"},
		{ID: "AR-219", Project: "The Hauz Garden", Customer: "Bpk. Wijaya", Type: "dp", Amount: 95, Aging: 0, Bucket: "current", SLA: "ok", Owner: "Rina", Next: "Termin DP sesuai jadwal"},
		{ID: "AR-244", Project: "Le Hauz Serpong", Customer: "Bpk. Iskandar", Type: "dp", Amount: 240, Aging: 108, Bucket: "d90", SLA: "overdue", Owner: "Sari", Next: "Opsi pembatalan / forfeit DP"},
	}
}

func seedPriorityMeta() []domain.MetaItem {
	return []domain.MetaItem{
		{Key: "high", Label: "Prioritas Tinggi", Tone: "red", Note: "Jatuh tempo / berdampak ke progres proyek"},
		{Key: "med", Label: "Prioritas Sedang", Tone: "orange", Note: "Jadwalkan dalam siklus bayar mingguan"},
		{Key: "low", Label: "Prioritas Rendah", Tone: "neutral", Note: "Masih dalam tenggat — bayar sesuai termin"},
	}
}

func seedPayables() []domain.Payable {
	return []domain.Payable{
		{ID: "AP-512", Vendor: "PT Bangun Cipta", Project: "Greenpark Aurora", Category: "termin", Amount: 1850, DueDays: -3, Priority: "high", Status: "overdue", Note: "Termin 3 — retensi 5% ditahan"},
		{ID: "AP-508", Vendor: "CV Karya Mandiri", Project: "Le Hauz Cibubur", Category: "termin", Amount: 1240, DueDays: 2, Priority: "high", Status: "due", Note: "Tahan sebagian — kaitkan ke recovery"},
		{ID: "AP-505", Vendor: "PT Sumber Material", Project: "Z Hauz Limo", Category: "material", Amount: 760, DueDays: 5, Priority: "med", Status: "due", Note: "Harga besi naik — verifikasi PO"},
		{ID: "AP-501", Vendor: "PT Graha Selaras", Project: "The Hauz Premiere", Category: "termin", Amount: 980, DueDays: 9, Priority: "med", Status: "ok", Note: "Sesuai progres — siap bayar"},
		{ID: "AP-498", Vendor: "Mandor Pool A", Project: "Le Hauz Serpong", Category: "upah", Amount: 320, DueDays: 1, Priority: "high", Status: "due", Note: "Upah mingguan — jangan tertunda"},
		{ID: "AP-494", Vendor: "PLN / PDAM", Project: "—", Category: "overhead", Amount: 145, DueDays: 12, Priority: "low", Status: "ok", Note: "Utilitas kantor & site"},
		{ID: "AP-490", Vendor: "PT Mitra Konstruksi", Project: "Greenpark Aurora", Category: "termin", Amount: 540, DueDays: 18, Priority: "low", Status: "ok", Note: "Termin 2 — menunggu opname"},
	}
}

func seedFacilities() []domain.Facility {
	return []domain.Facility{
		{Name: "Kredit Investasi — Bank A", Type: "KI", Plafond: 60000, Used: 38500, Rate: 9.5, Tenor: "60 bln", Status: domain.StatusGreen},
		{Name: "Modal Kerja — Bank A", Type: "KMK", Plafond: 25000, Used: 21800, Rate: 10.25, Tenor: "12 bln", Status: domain.StatusYellow},
		{Name: "KPR Pipeline — Bank B", Type: "KPR", Plafond: 48000, Used: 31200, Rate: 8.75, Tenor: "per akad", Status: domain.StatusGreen},
		{Name: "Ekuitas / Setor Modal", Type: "Equity", Plafond: 40000, Used: 40000, Rate: 0, Tenor: "—", Status: domain.StatusGreen},
	}
}

func seedCostStructure() []domain.CostCategory {
	return []domain.CostCategory{
		{Name: "Lahan", Budget: 42000, Actual: 42000},
		{Name: "Konstruksi", Budget: 74000, Actual: 58200},
		{Name: "Marketing", Budget: 9800, Actual: 7400},
		{Name: "Perizinan", Budget: 6200, Actual: 5100},
		{Name: "Overhead", Budget: 11400, Actual: 9600},
		{Name: "Biaya Bunga", Budget: 8600, Actual: 7900},
	}
}

func seedTreasury() domain.Treasury {
	return domain.Treasury{
		CashOnHand:     42300,
		RestrictedCash: 6800,
		MonthlyBurn:    9500,
	}
}

func seedAIInsights() []domain.AIInsight {
	return []domain.AIInsight{
		{Type: "Cash Flow Risk", Tone: "red", Text: "Le Hauz Cibubur: koleksi tertahan Rp 660 jt > 90 hari sementara biaya hampir habis budget — tekanan kas nyata.", Icon: "cash"},
		{Type: "Margin Erosion", Tone: "orange", Text: "Margin Serpong turun ke 19% akibat penagihan lambat & biaya overhead; di bawah ambang 22%.", Icon: "trend"},
		{Type: "Collection Pattern", Tone: "orange", Text: "3 dari 4 piutang macet berasal dari skema DP/cash bertahap — perketat syarat akad.", Icon: "receipt"},
		{Type: "Facility Usage", Tone: "yellow", Text: "Modal kerja Bank A terpakai 87% — ruang tarik tersisa Rp 3,2 M untuk 12 bulan.", Icon: "bank"},
		{Type: "Payables Due", Tone: "red", Text: "Termin PT Bangun Cipta Rp 1,85 M telah jatuh tempo — risiko stop kerja jika tidak dibayar 48 jam.", Icon: "alert"},
		{Type: "Recommendation", Tone: "green", Text: "Prioritaskan tagih DP macet Cibubur, tahan termin vendor non-kritis, jaga runway > 4 bulan.", Icon: "rec"},
	}
}

func seedDecisions() []domain.Decision {
	return []domain.Decision{
		{Role: "CEO", Tone: "red", Text: "Setujui audit biaya Le Hauz Cibubur — cost overrun"},
		{Role: "Dirkeu", Tone: "orange", Text: "Tahan termin vendor non-kritis, jaga kas > Rp 35 M"},
		{Role: "Collection", Tone: "red", Text: "Eskalasi legal piutang DP macet > 90 hari (Rp 660 jt)"},
		{Role: "Treasury", Tone: "navy", Text: "Jadwalkan ulang tarikan KMK — utilisasi 87%"},
		{Role: "Sales", Tone: "orange", Text: "Perketat syarat DP & verifikasi KPR sebelum akad"},
		{Role: "Procurement", Tone: "navy", Text: "Renegosiasi harga besi — naik 8% dari RAB Limo"},
	}
}

func seedCashflowTrend() []domain.CashflowPoint {
	return []domain.CashflowPoint{
		{Period: "Jan", Inflow: 8200, Outflow: 7400},
		{Period: "Feb", Inflow: 11400, Outflow: 9100},
		{Period: "Mar", Inflow: 9800, Outflow: 10200},
		{Period: "Apr", Inflow: 13600, Outflow: 9800},
		{Period: "Mei", Inflow: 12100, Outflow: 11600},
		{Period: "Jun", Inflow: 14800, Outflow: 10400},
		{Period: "Jul", Inflow: 13200, Outflow: 12800},
		{Period: "Agu", Inflow: 15600, Outflow: 11200},
	}
}

// seedUsers creates the default accounts. Change these immediately in any real
// deployment. Default credentials: admin/admin123 and viewer/viewer123.
func seedUsers() []storeUser {
	mk := func(id, username, name string, role domain.Role, password string) storeUser {
		salt := passwd.NewSalt()
		return storeUser{
			ID:           id,
			Username:     username,
			Name:         name,
			Role:         role,
			Salt:         salt,
			PasswordHash: passwd.Hash(password, salt),
		}
	}
	return []storeUser{
		mk("usr-admin", "admin", "Administrator Finance", domain.RoleAdmin, "admin123"),
		mk("usr-viewer", "viewer", "Viewer", domain.RoleViewer, "viewer123"),
	}
}

func seedKPITable() []domain.KPI {
	return []domain.KPI{
		{No: 1, KPI: "Revenue Achievement", Def: "Realisasi penjualan vs target", PIC: "Dirkeu", Upd: "Bulanan", Green: "≥95%", Yellow: "85–94%", Red: "<85%", Val: "92%", State: "yellow"},
		{No: 2, KPI: "Gross Margin", Def: "Margin kotor seluruh proyek", PIC: "Dirkeu", Upd: "Bulanan", Green: "≥25%", Yellow: "20–24%", Red: "<20%", Val: "23%", State: "yellow"},
		{No: 3, KPI: "Net Profit Margin", Def: "Laba bersih / pendapatan", PIC: "Dirkeu", Upd: "Bulanan", Green: "≥15%", Yellow: "10–14%", Red: "<10%", Val: "13%", State: "yellow"},
		{No: 4, KPI: "Collection Rate", Def: "Kas tertagih / nilai kontrak", PIC: "Collection", Upd: "Mingguan", Green: "≥90%", Yellow: "75–89%", Red: "<75%", Val: "58%", State: "red"},
		{No: 5, KPI: "AR Overdue >90", Def: "Piutang macet di atas 90 hari", PIC: "Collection", Upd: "Mingguan", Green: "0", Yellow: "1", Red: ">1", Val: "2", State: "red"},
		{No: 6, KPI: "DSO (Days Sales Outstanding)", Def: "Rata-rata umur piutang", PIC: "Collection", Upd: "Bulanan", Green: "≤30", Yellow: "31–60", Red: ">60", Val: "44 hari", State: "yellow"},
		{No: 7, KPI: "AP Overdue", Def: "Hutang lewat jatuh tempo", PIC: "Treasury", Upd: "Mingguan", Green: "0", Yellow: "1–2", Red: ">2", Val: "1", State: "yellow"},
		{No: 8, KPI: "DPO (Days Payable Outstanding)", Def: "Rata-rata umur hutang", PIC: "Treasury", Upd: "Bulanan", Green: "30–45", Yellow: "46–60", Red: ">60", Val: "38 hari", State: "green"},
		{No: 9, KPI: "Cash Runway", Def: "Bulan kas bertahan (burn)", PIC: "Treasury", Upd: "Bulanan", Green: "≥6", Yellow: "3–5", Red: "<3", Val: "4.5 bln", State: "yellow"},
		{No: 10, KPI: "Budget Absorption", Def: "Serapan biaya vs RAB", PIC: "Dirkeu", Upd: "Bulanan", Green: "≤progres", Yellow: "+1–5%", Red: ">+5%", Val: "68%", State: "yellow"},
		{No: 11, KPI: "Cost Overrun", Def: "Proyek melebihi RAB", PIC: "Dirkeu", Upd: "Bulanan", Green: "0", Yellow: "1", Red: ">1", Val: "1", State: "yellow"},
		{No: 12, KPI: "Facility Utilization", Def: "Pemakaian fasilitas bank", PIC: "Treasury", Upd: "Bulanan", Green: "≤75%", Yellow: "76–90%", Red: ">90%", Val: "maks 87%", State: "yellow"},
		{No: 13, KPI: "Debt Service Coverage", Def: "Kemampuan bayar cicilan (DSCR)", PIC: "Dirkeu", Upd: "Bulanan", Green: "≥1.5x", Yellow: "1.2–1.49x", Red: "<1.2x", Val: "1.4x", State: "yellow"},
		{No: 14, KPI: "Opex Ratio", Def: "Biaya operasional / pendapatan", PIC: "Dirkeu", Upd: "Bulanan", Green: "≤8%", Yellow: "9–12%", Red: ">12%", Val: "9.5%", State: "yellow"},
		{No: 15, KPI: "Invoice Accuracy", Def: "Akurasi tagihan & dokumen", PIC: "Finance Adm", Upd: "Mingguan", Green: "≥98%", Yellow: "95–97%", Red: "<95%", Val: "96%", State: "yellow"},
	}
}

func seedTriggers() []domain.Trigger {
	return []domain.Trigger{
		{Cond: "Cash runway menipis", Thr: "<3 bulan", Status: "red", PIC: "Treasury", Act: "Tahan belanja non-kritis + tarik fasilitas", Esc: "Dirkeu/CEO"},
		{Cond: "Collection rate rendah", Thr: "<75%", Status: "red", PIC: "Collection", Act: "Gerak penagihan intensif 7 hari", Esc: "Dirkeu"},
		{Cond: "Piutang macet", Thr: ">90 hari", Status: "red", PIC: "Collection", Act: "Surat somasi / restrukturisasi", Esc: "Legal/Dirkeu"},
		{Cond: "Margin proyek turun", Thr: "<20%", Status: "yellow", PIC: "Dirkeu", Act: "Review harga & struktur biaya", Esc: "CEO"},
		{Cond: "Cost overrun", Thr: ">RAB +5%", Status: "red", PIC: "Dirkeu", Act: "Audit biaya + freeze PO baru", Esc: "CEO"},
		{Cond: "Hutang jatuh tempo", Thr: "Lewat tempo", Status: "red", PIC: "Treasury", Act: "Bayar prioritas / negosiasi tempo", Esc: "Dirkeu"},
		{Cond: "Utilisasi fasilitas tinggi", Thr: ">90%", Status: "yellow", PIC: "Treasury", Act: "Negosiasi tambah plafond", Esc: "Dirkeu"},
		{Cond: "DSCR melemah", Thr: "<1.2x", Status: "red", PIC: "Dirkeu", Act: "Restruktur cicilan + jaga arus kas", Esc: "CEO"},
		{Cond: "DP belum lunas", Thr: ">30 hari", Status: "yellow", PIC: "Sales/Collection", Act: "Tahan akad sampai DP beres", Esc: "Dirkeu"},
		{Cond: "Selisih kas vs buku", Thr: ">Rp 50 jt", Status: "red", PIC: "Finance Adm", Act: "Rekonsiliasi + audit internal", Esc: "Dirkeu"},
		{Cond: "Beban bunga naik", Thr: ">RAB bunga", Status: "yellow", PIC: "Treasury", Act: "Evaluasi refinancing", Esc: "Dirkeu"},
		{Cond: "Fraud / penyimpangan", Thr: "Indikasi terdeteksi", Status: "crisis", PIC: "Dirkeu", Act: "Bekukan akses + investigasi", Esc: "CEO"},
	}
}
