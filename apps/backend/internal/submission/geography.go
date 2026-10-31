package submission

/** Catálogo del formulario, idéntico a src/data/ecuadorGeo.ts; T7 lo persistirá. */
var ecuadorProvinces = map[string][]string{
	"Azuay":                          {"Cuenca", "Chordeleg", "El Pan", "Girón", "Guachapala", "Gualaceo", "Nabón", "Oña", "Paute", "Ponce Enríquez", "Pucará", "San Fernando", "Santa Isabel", "Sevilla de Oro", "Sigsig"},
	"Bolívar":                        {"Guaranda", "Caluma", "Chillanes", "Chimbo", "Echeandía", "Las Naves", "San Miguel"},
	"Cañar":                          {"Azogues", "Biblián", "Cañar", "Déleg", "El Tambo", "La Troncal", "Suscal"},
	"Carchi":                         {"Tulcán", "Bolívar", "Espejo", "Mira", "Montúfar", "San Pedro de Huaca"},
	"Chimborazo":                     {"Riobamba", "Alausi", "Chambo", "Chunchi", "Colta", "Cumandá", "Guamote", "Guano", "Pallatanga", "Penipe"},
	"Cotopaxi":                       {"Latacunga", "La Maná", "Pangua", "Pujilí", "Salcedo", "Saquisilí", "Sigchos"},
	"El Oro":                         {"Machala", "Arenillas", "Atahualpa", "Balsas", "Chilla", "El Guabo", "Huaquillas", "Las Lajas", "Marcabelí", "Pasaje", "Piñas", "Portovelo", "Santa Rosa", "Zaruma"},
	"Esmeraldas":                     {"Esmeraldas", "Atacames", "Eloy Alfaro", "Muisne", "Quinindé", "Rioverde", "San Lorenzo"},
	"Galápagos":                      {"San Cristóbal", "Isabela", "Santa Cruz"},
	"Guayas":                         {"Guayaquil", "Alfredo Baquerizo Moreno (Juján)", "Balao", "Balzar", "Colimes", "Daule", "Durán", "El Empalme", "El Triunfo", "General Antonio Elizalde (Bucay)", "Isidro Ayora", "Lomas de Sargentillo", "Marcelino Maridueña", "Milagro", "Naranjal", "Naranjito", "Nobol", "Palestina", "Pedro Carbo", "Playas", "Salitre", "Samborondón", "Santa Lucía", "Simón Bolívar", "Yaguachi"},
	"Imbabura":                       {"Ibarra", "Antonio Ante", "Cotacachi", "Otavalo", "Pimampiro", "San Miguel de Urcuquí"},
	"Loja":                           {"Loja", "Calvas", "Catamayo", "Celica", "Chaguarpamba", "Espíndola", "Gonzanamá", "Macará", "Olmedo", "Paltas", "Pindal", "Puyango", "Quilanga", "Saraguro", "Sozoranga", "Zapotillo"},
	"Los Ríos":                       {"Babahoyo", "Baba", "Buena Fe", "Mocache", "Montalvo", "Palenque", "Puebloviejo", "Quevedo", "Quinsaloma", "Urdaneta", "Valencia", "Ventanas", "Vinces"},
	"Manabí":                         {"Portoviejo", "Bolívar", "Chone", "El Carmen", "Flavio Alfaro", "Jama", "Jaramijó", "Jipijapa", "Junín", "Manta", "Montecristi", "Olmedo", "Paján", "Pedernales", "Pichincha", "Puerto López", "Rocafuerte", "San Vicente", "Santa Ana", "Sucre (Bahía)", "Tosagua", "Veinticuatro de Mayo"},
	"Morona Santiago":                {"Morona (Macas)", "Gualaquiza", "Huamboya", "Limón Indanza", "Logroño", "Pablo Sexto", "Palora", "San Juan Bosco", "Santiago", "Sucúa", "Taisha", "Tiwintza"},
	"Napo":                           {"Tena", "Archidona", "Carlos Julio Arosemena Tola", "El Chaco", "Quijos (Baeza)"},
	"Orellana":                       {"Francisco de Orellana (El Coca)", "Aguarico", "La Joya de los Sachas", "Loreto"},
	"Pastaza":                        {"Pastaza (Puyo)", "Arajuno", "Mera", "Santa Clara"},
	"Pichincha":                      {"Quito", "Cayambe", "Mejía (Machachi)", "Pedro Moncayo (Tabacundo)", "Pedro Vicente Maldonado", "Puerto Quito", "Rumiñahui (Sangolquí)", "San Miguel de Los Bancos"},
	"Santa Elena":                    {"Santa Elena", "La Libertad", "Salinas"},
	"Santo Domingo de los Tsáchilas": {"Santo Domingo", "La Concordia"},
	"Sucumbíos":                      {"Lago Agrio (Nueva Loja)", "Cascales", "Cuyabeno", "Gonzalo Pizarro", "Putumayo", "Shushufindi", "Sucumbíos"},
	"Tungurahua":                     {"Ambato", "Baños de Agua Santa", "Cevallos", "Mocha", "Patate", "Pelileo", "Píllaro", "Quero", "Tisaleo"},
	"Zamora Chinchipe":               {"Zamora", "Centinela del Cóndor", "Chinchipe", "El Pangui", "Nangaritza", "Palanda", "Paquisha", "Yacuambi", "Yantzaza"},
}
