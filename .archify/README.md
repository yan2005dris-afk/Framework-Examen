# Diagramas Archify - Especificación Funcional BIESS

Diagramas interactivos generados con **Archify 3.2.0** que visualizan los flujos principales del requerimiento funcional para el **Concurso Público de Méritos y Oposición - Directorio del Banco del IESS**.

---

## 📊 Diagramas Disponibles

### 1️⃣ EF-11: Registro y Autenticación (01-auth-registration.html)

**Flujo completo:**
- Acceso a "Registrarse"
- Validación de cédula con Registro Civil (fecha expedición O código dactilar)
- Intento de registro con límite 3 intentos, lockout 10 minutos si falla
- Login con cédula + contraseña
- Recuperación de contraseña con revalidación de identidad

**Actores:** Postulante, Plataforma BIESS, Registro Civil, API Backend

**Especificaciones:**
- Max 3 intentos de registro
- Lockout 10 min tras fallo
- Validación interop Registro Civil Ecuador
- Error genérico en login incorrecto
- Link funcional "¿Olvidaste tu contraseña?"

---

### 2️⃣ EF-12: Formulario de Postulación (02-application-eligibility.html)

**Flujo completo:**
- Auto-carga datos personales desde Registro Civil
- Solicitud de domiciliario y contacto
- Selección de sector (Afiliado, Jubilado, Empleador)
- **Validación de elegibilidad:**
  - Afiliado → consulta IESS: ¿es afiliado?
  - Jubilado → consulta IESS: ¿acredita jubilación?
  - Empleador → consulta IESS: ¿es empleador?
  - Si NO cumple → cierre automático del formulario
- Carga de documentos respaldo (max 4 MB PDF)
- Formación académica (múltiple, con níveis)
- Experiencia laboral (auto-calcula años/meses/días)
- Publicaciones e investigaciones (opcional)
- Capacitaciones (Curso/Seminario/Taller)
- 13 preguntas de inhabilidades (Si/No)
- Consentimiento (5 declaraciones legales)
- Asignación de código único
- Envío de PDF a email postulante

**Actores:** Postulante, Plataforma BIESS, Registro Civil, IESS (Historial Laboral), API Backend

**Especificaciones:**
- Información personal auto-cargada
- Validaciones automáticas por sector
- Gestión de documentos: max 2-4 MB PDF
- Cálculo automático de experiencia
- Código de postulación único
- PDF generado automáticamente
- Trazabilidad completa y auditable

---

### 3️⃣ EF-14: Evaluación Teórica (03-theoretical-evaluation.html)

**Flujo completo:**
- Administrador carga banco de preguntas (múltiple opción)
- Selecciona postulantes elegibles (pasaron Fase Méritos)
- Marca estado: "Evaluación Técnica"
- Define fecha y hora exacta del examen
- Postulante accede el día/hora programados
- **Algoritmo de aleatoriedad:**
  - Extrae N preguntas al azar del banco
  - Orden diferente para cada postulante
  - Imposible que dos exámenes sean idénticos
- Pregunta por pregunta (una a la vez)
- Reloj cuenta regresiva visible
- Bloqueos de seguridad: no "Atrás", no Copy/Paste
- Si temporizador = 0 → cierre automático
- Click "Finalizar y Calificar" → envío de respuestas
- Comparación automática con matriz de respuestas correctas
- Cálculo matemático de nota
- Almacenamiento en expediente digital
- Generación de PDF con resultado
- Notificación automática por email

**Actores:** Administrador (SDNGTH), Plataforma BIESS, Base de Datos, Postulante, Motor Evaluación

**Especificaciones:**
- Aleatoriedad garantizada
- Temporizador configurable
- Bloqueos anti-fraude (Atrás, Copy/Paste)
- Timeout fuerza cierre automático
- Calificación instantánea
- PDF generado inmediatamente
- Trazabilidad completa en módulo administración

---

## 🎯 Cómo Usar

### Visualizar Diagramas

1. **Abrir INDEX.html** en navegador web
   - Menú interactivo con enlaces a los 3 diagramas
   - Resumen de cada flujo
   - Estadísticas generales

2. **Navegar cada diagrama:**
   - Click en el enlace "Ver Diagrama →"
   - Se abre en nueva pestaña
   - Diagrama interactivo con animación `trace`

### Características de los Diagramas

- ✅ **Interactivos:** Animaciones que trazan el flujo
- 📱 **Responsive:** Se adaptan al tamaño de pantalla
- 🎨 **Esquema visual:** Colores para diferenciar actores y tipos de acción
- 🔖 **Secciones:** Segmentos que organizan etapas del flujo
- 💬 **Tarjetas info:** Puntos clave (validaciones, seguridad, etc)

---

## 📁 Archivos Incluidos

```
.archify/
├── INDEX.html                          # Página principal (menú)
├── 01-auth-registration.html          # Diagrama EF-11 (827 KB)
├── 02-application-eligibility.html    # Diagrama EF-12 (824 KB)
├── 03-theoretical-evaluation.html     # Diagrama EF-14 (828 KB)
├── 01-auth-registration.json          # Fuente JSON EF-11
├── 02-application-eligibility.json    # Fuente JSON EF-12
├── 03-theoretical-evaluation.json     # Fuente JSON EF-14
├── 04-system-architecture.json        # Fuente JSON (arquitectura, no renderizado)
└── README.md                           # Este archivo
```

---

## 🔧 Generación de Diagramas

Renderizados con **Archify 3.2.0:**

```bash
# Diagrama 1
/home/andy/.local/archify/archify/bin/archify.mjs render sequence \
  01-auth-registration.json \
  01-auth-registration.html \
  --quality showcase

# Diagrama 2
/home/andy/.local/archify/archify/bin/archify.mjs render sequence \
  02-application-eligibility.json \
  02-application-eligibility.html \
  --quality showcase

# Diagrama 3
/home/andy/.local/archify/archify/bin/archify.mjs render sequence \
  03-theoretical-evaluation.json \
  03-theoretical-evaluation.html \
  --quality showcase
```

---

## 📋 Especificaciones Técnicas

| Aspecto | Detalle |
|--------|---------|
| **Tipo de Diagrama** | Sequence (flujo de interacciones) |
| **Generador** | Archify 3.2.0 |
| **Formato Entrada** | JSON (schema v1) |
| **Formato Salida** | HTML interactivo (PNG embedded) |
| **Animación** | trace (sigue el flujo) |
| **Calidad** | showcase (máxima fidelidad) |
| **Tamaño por Diagrama** | ~825 KB (includes PNG, CSS, JS) |

---

## 🎓 Mapeo a Requerimiento Funcional

| Diagrama | Especificación | Referencia |
|----------|---|---|
| 01-auth-registration | EF-11 | Registro y autenticación |
| 02-application-eligibility | EF-12 | Formulario de postulación |
| 03-theoretical-evaluation | EF-14 | Evaluación teórica |

Documentación: `docs/extras/requerimiento.pdf` (v1.4, 26/3/2026)

---

## ✅ Validación

Todos los diagramas:
- ✓ Schema JSON válido
- ✓ Renderizados sin errores
- ✓ Interactivos en navegador web
- ✓ Responsive (adaptable a pantallas)
- ✓ Trazabilidad completa de flujos

---

## 📞 Contacto

**Instituto Ecuatoriano de Seguridad Social (IESS)**  
Subdirección Nacional de Gestión de Talento Humano  
Dirección Nacional de Tecnologías de la Información

Documento: Requerimiento Funcional v1.4  
Fecha: 26 de marzo de 2026  
Código: (no especificado)

---

Generado: 3 de noviembre de 2026  
Herramienta: Archify 3.2.0  
Usuario: andy
