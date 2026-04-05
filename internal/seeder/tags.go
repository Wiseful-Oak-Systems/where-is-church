package seeder

import (
	"log"

	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

// SeedDefaultTags inserts the default tag set if no tags exist yet.
func SeedDefaultTags(db *gorm.DB) {
	var count int64
	db.Model(&models.Tag{}).Count(&count)
	if count > 0 {
		return
	}

	log.Println("Seeding default tags...")
	tags := defaultTags()
	for _, tag := range tags {
		db.Create(&tag)
	}
	log.Printf("Seeded %d tags", len(tags))
}

func defaultTags() []models.Tag {
	return []models.Tag{
		// Liturgical Rite
		{Key: "latin_mass", LabelEN: "Latin Mass (TLM)", LabelPT: "Missa em Latim (Forma Extraordinária)", Category: "rite", Icon: "🕊️"},
		{Key: "novus_ordo", LabelEN: "Novus Ordo", LabelPT: "Novus Ordo (Forma Ordinária)", Category: "rite", Icon: "⛪"},
		{Key: "maronite", LabelEN: "Maronite Rite", LabelPT: "Rito Maronita", Category: "rite", Icon: "🌿"},
		{Key: "byzantine", LabelEN: "Byzantine Rite", LabelPT: "Rito Bizantino", Category: "rite", Icon: "☦️"},
		{Key: "melkite", LabelEN: "Melkite Rite", LabelPT: "Rito Melquita", Category: "rite", Icon: "🕯️"},
		{Key: "syro_malabar", LabelEN: "Syro-Malabar Rite", LabelPT: "Rito Siro-Malabar", Category: "rite"},
		{Key: "ordinariate", LabelEN: "Anglican Ordinariate", LabelPT: "Ordinariato Anglicano", Category: "rite"},

		// Catholic Societies / Orders
		{Key: "fsspx", LabelEN: "FSSPX (Lefebvre)", LabelPT: "FSSPX (Fraternidade São Pio X)", Category: "society", Icon: "✠"},
		{Key: "fssp", LabelEN: "FSSP (Priestly Fraternity)", LabelPT: "FSSP (Fraternidade Sacerdotal São Pedro)", Category: "society", Icon: "✠"},
		{Key: "icksp", LabelEN: "ICKSP (Institute of Christ the King)", LabelPT: "ICRSP (Instituto Cristo Rei Sumo Sacerdote)", Category: "society"},
		{Key: "opus_dei", LabelEN: "Opus Dei", LabelPT: "Opus Dei", Category: "society"},
		{Key: "dominican", LabelEN: "Dominican", LabelPT: "Dominicano", Category: "society"},
		{Key: "franciscan", LabelEN: "Franciscan", LabelPT: "Franciscano", Category: "society"},
		{Key: "jesuit", LabelEN: "Jesuit", LabelPT: "Jesuíta", Category: "society"},
		{Key: "benedictine", LabelEN: "Benedictine", LabelPT: "Beneditino", Category: "society"},
		{Key: "carmelite", LabelEN: "Carmelite", LabelPT: "Carmelita", Category: "society"},

		// Sacrament Availability
		{Key: "daily_confession", LabelEN: "Daily Confession", LabelPT: "Confissão Diária", Category: "sacrament", Icon: "🙏"},
		{Key: "perpetual_adoration", LabelEN: "Perpetual Adoration", LabelPT: "Adoração Perpétua", Category: "sacrament", Icon: "🕊️"},
		{Key: "daily_mass", LabelEN: "Daily Mass", LabelPT: "Missa Diária", Category: "sacrament"},
		{Key: "baptism_prep", LabelEN: "Baptism Preparation", LabelPT: "Preparação para Batismo", Category: "sacrament"},
		{Key: "marriage_prep", LabelEN: "Marriage Preparation", LabelPT: "Preparação para Casamento", Category: "sacrament"},
		{Key: "rcia", LabelEN: "RCIA / Catechumenate", LabelPT: "Catecumenato / RICA", Category: "sacrament"},

		// Family & Accessibility
		{Key: "children_friendly", LabelEN: "Children Friendly", LabelPT: "Acolhe Crianças", Category: "family", Icon: "👶"},
		{Key: "cry_room", LabelEN: "Cry Room", LabelPT: "Sala de Choro", Category: "family", Icon: "🍼"},
		{Key: "sunday_school", LabelEN: "Sunday School / Catechesis", LabelPT: "Catequese / Escola Dominical", Category: "family"},
		{Key: "youth_group", LabelEN: "Youth Group", LabelPT: "Grupo de Jovens", Category: "family"},
		{Key: "wheelchair_accessible", LabelEN: "Wheelchair Accessible", LabelPT: "Acessível para Cadeirantes", Category: "accessibility", Icon: "♿"},
		{Key: "hearing_loop", LabelEN: "Hearing Loop", LabelPT: "Anel de Indução Magnética", Category: "accessibility"},
		{Key: "sign_language", LabelEN: "Sign Language Mass", LabelPT: "Missa em Libras", Category: "accessibility"},

		// Languages
		{Key: "mass_portuguese", LabelEN: "Mass in Portuguese", LabelPT: "Missa em Português", Category: "language"},
		{Key: "mass_latin", LabelEN: "Mass in Latin", LabelPT: "Missa em Latim", Category: "language"},
		{Key: "mass_english", LabelEN: "Mass in English", LabelPT: "Missa em Inglês", Category: "language"},
		{Key: "mass_spanish", LabelEN: "Mass in Spanish", LabelPT: "Missa em Espanhol", Category: "language"},

		// Facilities
		{Key: "parking", LabelEN: "Parking Available", LabelPT: "Estacionamento", Category: "facilities", Icon: "🅿️"},
		{Key: "air_conditioning", LabelEN: "Air Conditioning", LabelPT: "Ar Condicionado", Category: "facilities", Icon: "❄️"},
		{Key: "live_streaming", LabelEN: "Live Streaming", LabelPT: "Transmissão ao Vivo", Category: "facilities", Icon: "📺"},
		{Key: "gift_shop", LabelEN: "Gift Shop / Bookstore", LabelPT: "Loja / Livraria", Category: "facilities"},
		{Key: "cafeteria", LabelEN: "Cafeteria / Social Hall", LabelPT: "Cafeteria / Salão Social", Category: "facilities"},

		// Community
		{Key: "bible_study", LabelEN: "Bible Study", LabelPT: "Estudo Bíblico", Category: "community"},
		{Key: "choir", LabelEN: "Choir", LabelPT: "Coral", Category: "community", Icon: "🎵"},
		{Key: "social_outreach", LabelEN: "Social Outreach", LabelPT: "Pastoral Social", Category: "community"},
		{Key: "food_bank", LabelEN: "Food Bank", LabelPT: "Banco de Alimentos", Category: "community"},
		{Key: "rosary_group", LabelEN: "Rosary Group", LabelPT: "Grupo do Rosário", Category: "community", Icon: "📿"},
		{Key: "legion_of_mary", LabelEN: "Legion of Mary", LabelPT: "Legião de Maria", Category: "community"},

		// Church Type
		{Key: "cathedral", LabelEN: "Cathedral", LabelPT: "Catedral", Category: "type", Icon: "🏛️"},
		{Key: "basilica", LabelEN: "Basilica", LabelPT: "Basílica", Category: "type"},
		{Key: "shrine", LabelEN: "Shrine / Sanctuary", LabelPT: "Santuário", Category: "type"},
		{Key: "chapel", LabelEN: "Chapel", LabelPT: "Capela", Category: "type"},
		{Key: "monastery", LabelEN: "Monastery / Convent", LabelPT: "Mosteiro / Convento", Category: "type"},

		// Additional from research
		{Key: "coptic", LabelEN: "Coptic Rite", LabelPT: "Rito Copta", Category: "rite"},
		{Key: "armenian", LabelEN: "Armenian Rite", LabelPT: "Rito Armênio", Category: "rite"},
		{Key: "legionaries", LabelEN: "Legionaries of Christ", LabelPT: "Legionários de Cristo", Category: "society"},
		{Key: "nursing_room", LabelEN: "Nursing Mothers Room", LabelPT: "Sala de Amamentação", Category: "family", Icon: "🤱"},
		{Key: "elevator", LabelEN: "Elevator Available", LabelPT: "Elevador Disponível", Category: "accessibility"},
		{Key: "accessible_restrooms", LabelEN: "Accessible Restrooms", LabelPT: "Banheiros Acessíveis", Category: "accessibility"},
		{Key: "anointing_sick", LabelEN: "Anointing of the Sick", LabelPT: "Unção dos Enfermos", Category: "sacrament"},
		{Key: "online_giving", LabelEN: "Online Donations", LabelPT: "Doações Online", Category: "facilities"},
		{Key: "bulletin", LabelEN: "Parish Bulletin", LabelPT: "Boletim Paroquial", Category: "facilities"},
		{Key: "women_group", LabelEN: "Women's Ministry", LabelPT: "Pastoral da Mulher", Category: "community"},
		{Key: "men_group", LabelEN: "Men's Ministry", LabelPT: "Pastoral do Homem", Category: "community"},
	}
}
