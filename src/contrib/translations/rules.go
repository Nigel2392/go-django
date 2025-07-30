package translations

type pluralRule struct {
	l int
	r string
}

var pluralRules = map[string]pluralRule{
	"af":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Afrikaans
	"ak":    {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Akan
	"sq":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Albanian
	"am":    {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Amharic
	"ar":    {6, "(n == 0) ? 0 : (n == 1) ? 1 : (n == 2) ? 2 : (n % 100 >= 3 && n % 100 <= 10) ? 3 : (n % 100 >= 11 && n % 100 <= 99) ? 4 : 5"},      // Arabic
	"an":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Aragonese
	"hy":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Armenian
	"as":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Assamese
	"asa":   {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Asu
	"az":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Azerbaijani
	"bal":   {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Baluchi
	"bm":    {1, "0"},                                                                                                                                // Bambara
	"eu":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Basque
	"be":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Belarusian
	"bem":   {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Bemba
	"bez":   {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Bena
	"bho":   {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Bhojpuri
	"brx":   {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Bodo
	"bs":    {3, "(n % 10 == 1 && n % 100 != 11) ? 0 : (n % 10 >= 2 && n % 10 <= 4 && (n % 100 < 12 || n % 100 > 14)) ? 1 : (n != 0) ? 2 : 3"},       // Bosnian
	"bg":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Bulgarian
	"my":    {1, "0"},                                                                                                                                // Burmese
	"yue":   {1, "0"},                                                                                                                                // Cantonese
	"ca":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Catalan
	"ceb":   {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Cebuano
	"ckb":   {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Central Kurdish	z
	"chr":   {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Cherokee
	"zh":    {1, "0"},                                                                                                                                // Chinese
	"hr":    {3, "(n % 10 == 1 && n % 100 != 11) ? 0 : (n % 10 >= 2 && n % 10 <= 4 && (n % 100 < 12 || n % 100 > 14)) ? 1 : (n != 0) ? 2 : 3"},       // Croatian
	"cs":    {3, "(n == 1) ? 0 : (n >= 2 && n <= 4) ? 1 : 2"},                                                                                        // Czech
	"da":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Danish
	"dv":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Divehi
	"doi":   {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Dogri
	"nl":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Dutch
	"en":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // English
	"eo":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Esperanto
	"et":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Estonian
	"pt_PT": {3, "(n == 0) ? 0 : (n == 1) ? 1 : (n != 0) ? 2 : 3"},                                                                                   // European Portuguese
	"ee":    {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Ewe
	"fil":   {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Filipino
	"fi":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Finnish
	"fr":    {2, "(n > 1) ? 1 : 0"},                                                                                                                  // French
	"lg":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Ganda
	"ka":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Georgian
	"de":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // German
	"el":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Greek
	"haw":   {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Hawaiian
	"he":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Hebrew
	"hi":    {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Hindi
	"hu":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Hungarian
	"is":    {2, "(n % 10 == 1 && n % 100 != 11) ? 0 : (n % 10 >= 2 && n % 10 <= 4 && (n % 100 < 12 || n % 100 > 14)) ? 1 : (n != 0) ? 2 : 3"},       // Icelandic
	"id":    {1, "0"},                                                                                                                                // Indonesian
	"ga":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Irish
	"it":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Italian
	"ja":    {1, "0"},                                                                                                                                // Japanese
	"jv":    {1, "0"},                                                                                                                                // Javanese
	"ko":    {1, "0"},                                                                                                                                // Korean
	"ku":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Kurdish
	"lv":    {3, "(n == 1) ? 0 : (n % 10 >= 2 && n % 10 <= 9 && (n % 100 < 11 || n % 100 > 19)) ? 1 : 2"},                                            // Latvian
	"lt":    {3, "(n == 1) ? 0 : (n % 10 == 1 && n % 100 != 11) ? 1 : 2"},                                                                            // Lithuanian
	"lb":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Luxembourgish
	"mk":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Macedonian
	"ms":    {1, "0"},                                                                                                                                // Malay
	"ml":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Malayalam
	"mt":    {5, "(n == 1) ? 0 : (n == 0 || n % 100 > 1 && n % 100 < 11) ? 1 : (n % 100 > 10 && n % 100 < 20) ? 2 : (n != 0) ? 3 : 4"},               // Maltese
	"mo":    {3, "(n % 10 == 1 && n % 100 != 11) ? 0 : (n % 10 >= 2 && n % 10 <= 4 && (n % 100 < 12 || n % 100 > 14)) ? 1 : (n != 0) ? 2 : 3"},       // Moldavian
	"mn":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Mongolian
	"ne":    {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Nepali
	"nso":   {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Northern Sotho
	"no":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Norwegian
	"or":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Odia
	"om":    {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Oromo
	"ps":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Pashto
	"fa":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Persian
	"sr":    {3, "(n % 10 == 1 && n % 100 != 11) ? 0 : (n % 10 >= 2 && n % 10 <= 4 && (n % 100 < 12 || n % 100 > 14)) ? 1 : (n != 0) ? 2 : 3"},       // Serbian
	"scn":   {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Sicilian
	"sk":    {3, "(n == 1) ? 0 : (n >= 2 && n <= 4) ? 1 : 2"},                                                                                        // Slovak
	"sl":    {4, "(n % 100 == 1) ? 0 : (n % 100 == 2) ? 1 : (n % 100 == 3 || n % 100 == 4) ? 2 : 3"},                                                 // Slovenian
	"so":    {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Somali
	"es":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Spanish
	"su":    {1, "0"},                                                                                                                                // Sundanese
	"sw":    {2, "(n > 1) ? 1 : 0"},                                                                                                                  // Swahili
	"sv":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Swedish
	"th":    {1, "0"},                                                                                                                                // Thai
	"bo":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Tibetan
	"tr":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Turkish
	"tk":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Turkmen
	"uk":    {3, "(n % 10 == 1 && n % 100 != 11) ? 0 : (n % 10 >= 2 && n % 10 <= 4 && (n % 100 < 12 || n % 100 > 14)) ? 1 : (n != 0) ? 2 : 3"},       // Ukrainian
	"ur":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Urdu
	"ug":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Uyghur
	"uz":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Uzbek
	"vi":    {1, "0"},                                                                                                                                // Vietnamese
	"cy":    {5, "(n == 0) ? 0 : (n == 1) ? 1 : (n == 2) ? 2 : (n == 3 || n == 8 || n == 11) ? 3 : (n == 6 || n == 7 || n == 9 || n == 10) ? 4 : 5"}, // Welsh
	"ji":    {1, "0"},                                                                                                                                // Jiddish
	"zu":    {2, "(n != 1) ? 1 : 0"},                                                                                                                 // Zulu
}
