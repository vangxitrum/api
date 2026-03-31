package ip_helper

import (
	"log/slog"

	custom_log "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/log"
)

var countryToContinentMapping map[string]string = map[string]string{
	"AF": "AS", // Afghanistan -> Asia
	"AL": "EU", // Albania -> Europe
	"DZ": "AF", // Algeria -> Africa
	"AS": "OC", // American Samoa -> Oceania
	"AD": "EU", // Andorra -> Europe
	"AO": "AF", // Angola -> Africa
	"AI": "NA", // Anguilla -> North America
	"AQ": "AN", // Antarctica -> Antarctica
	"AG": "NA", // Antigua and Barbuda -> North America
	"AR": "SA", // Argentina -> South America
	"AM": "AS", // Armenia -> Asia
	"AW": "NA", // Aruba -> North America
	"AU": "OC", // Australia -> Oceania
	"AT": "EU", // Austria -> Europe
	"AZ": "AS", // Azerbaijan -> Asia
	"BS": "NA", // Bahamas -> North America
	"BH": "AS", // Bahrain -> Asia
	"BD": "AS", // Bangladesh -> Asia
	"BB": "NA", // Barbados -> North America
	"BY": "EU", // Belarus -> Europe
	"BE": "EU", // Belgium -> Europe
	"BZ": "NA", // Belize -> North America
	"BJ": "AF", // Benin -> Africa
	"BM": "NA", // Bermuda -> North America
	"BT": "AS", // Bhutan -> Asia
	"BO": "SA", // Bolivia -> South America
	"BA": "EU", // Bosnia and Herzegovina -> Europe
	"BW": "AF", // Botswana -> Africa
	"BR": "SA", // Brazil -> South America
	"BN": "AS", // Brunei -> Asia
	"BG": "EU", // Bulgaria -> Europe
	"BF": "AF", // Burkina Faso -> Africa
	"BI": "AF", // Burundi -> Africa
	"KH": "AS", // Cambodia -> Asia
	"CM": "AF", // Cameroon -> Africa
	"CA": "NA", // Canada -> North America
	"CV": "AF", // Cape Verde -> Africa
	"KY": "NA", // Cayman Islands -> North America
	"CF": "AF", // Central African Republic -> Africa
	"TD": "AF", // Chad -> Africa
	"CL": "SA", // Chile -> South America
	"CN": "AS", // China -> Asia
	"CX": "AS", // Christmas Island -> Asia
	"CC": "AS", // Cocos Islands -> Asia
	"CO": "SA", // Colombia -> South America
	"KM": "AF", // Comoros -> Africa
	"CG": "AF", // Congo -> Africa
	"CD": "AF", // Democratic Republic of the Congo -> Africa
	"CK": "OC", // Cook Islands -> Oceania
	"CR": "NA", // Costa Rica -> North America
	"CI": "AF", // Côte d'Ivoire -> Africa
	"HR": "EU", // Croatia -> Europe
	"CU": "NA", // Cuba -> North America
	"CY": "AS", // Cyprus -> Asia
	"CZ": "EU", // Czech Republic -> Europe
	"DK": "EU", // Denmark -> Europe
	"DJ": "AF", // Djibouti -> Africa
	"DM": "NA", // Dominica -> North America
	"DO": "NA", // Dominican Republic -> North America
	"EC": "SA", // Ecuador -> South America
	"EG": "AF", // Egypt -> Africa
	"SV": "NA", // El Salvador -> North America
	"GQ": "AF", // Equatorial Guinea -> Africa
	"ER": "AF", // Eritrea -> Africa
	"EE": "EU", // Estonia -> Europe
	"ET": "AF", // Ethiopia -> Africa
	"FK": "SA", // Falkland Islands -> South America
	"FO": "EU", // Faroe Islands -> Europe
	"FJ": "OC", // Fiji -> Oceania
	"FI": "EU", // Finland -> Europe
	"FR": "EU", // France -> Europe
	"GF": "SA", // French Guiana -> South America
	"PF": "OC", // French Polynesia -> Oceania
	"TF": "AN", // French Southern Territories -> Antarctica
	"GA": "AF", // Gabon -> Africa
	"GM": "AF", // Gambia -> Africa
	"GE": "AS", // Georgia -> Asia
	"DE": "EU", // Germany -> Europe
	"GH": "AF", // Ghana -> Africa
	"GI": "EU", // Gibraltar -> Europe
	"GR": "EU", // Greece -> Europe
	"GL": "NA", // Greenland -> North America
	"GD": "NA", // Grenada -> North America
	"GP": "NA", // Guadeloupe -> North America
	"GU": "OC", // Guam -> Oceania
	"GT": "NA", // Guatemala -> North America
	"GG": "EU", // Guernsey -> Europe
	"GN": "AF", // Guinea -> Africa
	"GW": "AF", // Guinea-Bissau -> Africa
	"GY": "SA", // Guyana -> South America
	"HT": "NA", // Haiti -> North America
	"HM": "AN", // Heard Island and McDonald Islands -> Antarctica
	"VA": "EU", // Vatican -> Europe
	"HN": "NA", // Honduras -> North America
	"HK": "AS", // Hong Kong -> Asia
	"HU": "EU", // Hungary -> Europe
	"IS": "EU", // Iceland -> Europe
	"IN": "AS", // India -> Asia
	"ID": "AS", // Indonesia -> Asia
	"IR": "AS", // Iran -> Asia
	"IQ": "AS", // Iraq -> Asia
	"IE": "EU", // Ireland -> Europe
	"IM": "EU", // Isle of Man -> Europe
	"IL": "AS", // Israel -> Asia
	"IT": "EU", // Italy -> Europe
	"JM": "NA", // Jamaica -> North America
	"JP": "AS", // Japan -> Asia
	"JE": "EU", // Jersey -> Europe
	"JO": "AS", // Jordan -> Asia
	"KZ": "AS", // Kazakhstan -> Asia
	"KE": "AF", // Kenya -> Africa
	"KI": "OC", // Kiribati -> Oceania
	"KP": "AS", // North Korea -> Asia
	"KR": "AS", // South Korea -> Asia
	"KW": "AS", // Kuwait -> Asia
	"KG": "AS", // Kyrgyzstan -> Asia
	"LA": "AS", // Laos -> Asia
	"LV": "EU", // Latvia -> Europe
	"LB": "AS", // Lebanon -> Asia
	"LS": "AF", // Lesotho -> Africa
	"LR": "AF", // Liberia -> Africa
	"LY": "AF", // Libya -> Africa
	"LI": "EU", // Liechtenstein -> Europe
	"LT": "EU", // Lithuania -> Europe
	"LU": "EU", // Luxembourg -> Europe
	"MO": "AS", // Macau -> Asia
	"MK": "EU", // North Macedonia -> Europe
	"MG": "AF", // Madagascar -> Africa
	"MW": "AF", // Malawi -> Africa
	"MY": "AS", // Malaysia -> Asia
	"MV": "AS", // Maldives -> Asia
	"ML": "AF", // Mali -> Africa
	"MT": "EU", // Malta -> Europe
	"MH": "OC", // Marshall Islands -> Oceania
	"MR": "AF", // Mauritania -> Africa
	"MU": "AF", // Mauritius -> Africa
	"YT": "AF", // Mayotte -> Africa
	"MX": "NA", // Mexico -> North America
	"FM": "OC", // Micronesia -> Oceania
	"MD": "EU", // Moldova -> Europe
	"MC": "EU", // Monaco -> Europe
	"MN": "AS", // Mongolia -> Asia
	"ME": "EU", // Montenegro -> Europe
	"MS": "NA", // Montserrat -> North America
	"MA": "AF", // Morocco -> Africa
	"MZ": "AF", // Mozambique -> Africa
	"MM": "AS", // Myanmar -> Asia
	"NA": "AF", // Namibia -> Africa
	"NR": "OC", // Nauru -> Oceania
	"NP": "AS", // Nepal -> Asia
	"NL": "EU", // Netherlands -> Europe
	"NC": "OC", // New Caledonia -> Oceania
	"NZ": "OC", // New Zealand -> Oceania
	"NI": "NA", // Nicaragua -> North America
	"NE": "AF", // Niger -> Africa
	"NG": "AF", // Nigeria -> Africa
	"NU": "OC", // Niue -> Oceania
	"NF": "OC", // Norfolk Island -> Oceania
	"MP": "OC", // Northern Mariana Islands -> Oceania
	"NO": "EU", // Norway -> Europe
	"OM": "AS", // Oman -> Asia
	"PK": "AS", // Pakistan -> Asia
	"PW": "OC", // Palau -> Oceania
	"PS": "AS", // Palestine -> Asia
	"PA": "NA", // Panama -> North America
	"PG": "OC", // Papua New Guinea -> Oceania
	"PY": "SA", // Paraguay -> South America
	"PE": "SA", // Peru -> South America
	"PH": "AS", // Philippines -> Asia
	"PN": "OC", // Pitcairn Islands -> Oceania
	"PL": "EU", // Poland -> Europe
	"PT": "EU", // Portugal -> Europe
	"PR": "NA", // Puerto Rico -> North America
	"QA": "AS", // Qatar -> Asia
	"RE": "AF", // Réunion -> Africa
	"RO": "EU", // Romania -> Europe
	"RU": "EU", // Russia -> Europe (partially Asia)
	"RW": "AF", // Rwanda -> Africa
	"BL": "NA", // Saint Barthélemy -> North America
	"SH": "AF", // Saint Helena -> Africa
	"KN": "NA", // Saint Kitts and Nevis -> North America
	"LC": "NA", // Saint Lucia -> North America
	"MF": "NA", // Saint Martin -> North America
	"PM": "NA", // Saint Pierre and Miquelon -> North America
	"VC": "NA", // Saint Vincent and the Grenadines -> North America
	"WS": "OC", // Samoa -> Oceania
	"SM": "EU", // San Marino -> Europe
	"ST": "AF", // Sao Tome and Principe -> Africa
	"SA": "AS", // Saudi Arabia -> Asia
	"SN": "AF", // Senegal -> Africa
	"RS": "EU", // Serbia -> Europe
	"SC": "AF", // Seychelles -> Africa
	"SL": "AF", // Sierra Leone -> Africa
	"SG": "AS", // Singapore -> Asia
	"SX": "NA", // Sint Maarten -> North America
	"SK": "EU", // Slovakia -> Europe
	"SI": "EU", // Slovenia -> Europe
	"SB": "OC", // Solomon Islands -> Oceania
	"SO": "AF", // Somalia -> Africa
	"ZA": "AF", // South Africa -> Africa
	"GS": "AN", // South Georgia and the South Sandwich Islands -> Antarctica
	"SS": "AF", // South Sudan -> Africa
	"ES": "EU", // Spain -> Europe
	"LK": "AS", // Sri Lanka -> Asia
	"SD": "AF", // Sudan -> Africa
	"SR": "SA", // Suriname -> South America
	"SJ": "EU", // Svalbard and Jan Mayen -> Europe
	"SZ": "AF", // Eswatini (Swaziland) -> Africa
	"SE": "EU", // Sweden -> Europe
	"CH": "EU", // Switzerland -> Europe
	"SY": "AS", // Syria -> Asia
	"TW": "AS", // Taiwan -> Asia
	"TJ": "AS", // Tajikistan -> Asia
	"TZ": "AF", // Tanzania -> Africa
	"TH": "AS", // Thailand -> Asia
	"TL": "AS", // Timor-Leste -> Asia
	"TG": "AF", // Togo -> Africa
	"TK": "OC", // Tokelau -> Oceania
	"TO": "OC", // Tonga -> Oceania
	"TT": "NA", // Trinidad and Tobago -> North America
	"TN": "AF", // Tunisia -> Africa
	"TR": "EU", // Turkey -> Europe (partially Asia)
	"TM": "AS", // Turkmenistan -> Asia
	"TC": "NA", // Turks and Caicos Islands -> North America
	"TV": "OC", // Tuvalu -> Oceania
	"UG": "AF", // Uganda -> Africa
	"UA": "EU", // Ukraine -> Europe
	"AE": "AS", // United Arab Emirates -> Asia
	"GB": "EU", // United Kingdom -> Europe
	"US": "NA", // United States -> North America
	"UM": "OC", // United States Minor Outlying Islands -> Oceania
	"UY": "SA", // Uruguay -> South America
	"UZ": "AS", // Uzbekistan -> Asia
	"VU": "OC", // Vanuatu -> Oceania
	"VE": "SA", // Venezuela -> South America
	"VN": "AS", // Vietnam -> Asia
	"VG": "NA", // British Virgin Islands -> North America
	"VI": "NA", // U.S. Virgin Islands -> North America
	"WF": "OC", // Wallis and Futuna -> Oceania
	"EH": "AF", // Western Sahara -> Africa
	"YE": "AS", // Yemen -> Asia
	"ZM": "AF", // Zambia -> Africa
	"ZW": "AF", // Zimbabwe -> Africa
}

type IpInfo struct {
	IP          string  `json:"ip"           gorm:"ip"`
	CountryCode string  `json:"country_code" gorm:"country_code"`
	Region      string  `json:"region"       gorm:"region"`
	City        string  `json:"city"         gorm:"city"`
	Latitude    float64 `json:"latitude"     gorm:"latitude"`
	Longitude   float64 `json:"longitude"    gorm:"longitude"`
	Continent   string  `json:"continent"    gorm:"continent"`
}

type IpHelper interface {
	GetIpInfo(ip string) (*IpInfo, error)
}

type Option func(IpHelper)

func WithDebug(h IpHelper) {
	logger = slog.New(custom_log.NewHandler(&slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}
