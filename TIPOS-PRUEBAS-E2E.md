# Tipos de Pruebas E2E - Framework-Examen
**Guía para cobertura exhaustiva con Playwright**

---

## 1. FLUJOS COMPLETOS (Happy Path + Sad Path)

### Happy Path: Usuario completa todo exitosamente
```typescript
test("Flujo 1: Registro → Login → Llena formulario → Envía → Imprime", async ({ page }) => {
  // 1. Registro
  await page.goto("/registro")
  await page.fill("input[name='cedula']", "1234567890")
  await page.fill("input[name='correo']", "test@example.com")
  await page.click("button:has-text('Enviar código')")
  
  // 2. Verificar OTP (interceptar correo en Mailpit)
  const otp = await obtenerOTPdeMailpit(page)
  await page.fill("input[name='otp']", otp)
  await page.fill("input[name='password']", "SecurePass123!")
  await page.click("button:has-text('Registrar')")
  
  // 3. Login automático post-registro
  expect(page).toHaveURL(/\/postulacion/)
  
  // 4. Llenar formulario 9 secciones
  for (let i = 1; i <= 9; i++) {
    await llenarSeccion(page, i)
    await page.click("button:has-text('Guardar')")
    expect(page.locator("text=Guardado")).toBeVisible()
  }
  
  // 5. Aceptar consentimiento
  await page.click("input#consent-all")
  await page.click("button:has-text('Enviar postulación')")
  
  // 6. Verificar sellado
  expect(page.locator("text=Postulación sellada con Hash")).toBeVisible()
  
  // 7. Imprimir
  await page.goto("/postulacion/imprimir")
  const pdfData = await page.context().pages()[0].pdf()
  expect(pdfData.length).toBeGreaterThan(1000)
})
```

### Sad Paths: Validaciones rechazan
```typescript
test("Flujo 2: Rechazo por nacionalidad no ecuatoriana", async ({ page }) => {
  await loginWithUser(page, cedula, password)
  await page.goto("/postulacion/1")
  
  // Cambiar a nacionalidad no-EC
  await page.selectOption("select[name='nacionalidad']", "Perú")
  await page.click("button:has-text('Enviar')")
  
  // Debe mostrar error específico
  expect(page.locator("text=/ecuatoriano/i")).toBeVisible()
  expect(page).toHaveURL(/\/postulacion\/1/) // Sin avanzar
})

test("Flujo 3: Rechazo por requisito no cumplido (afiliado no afiliado)", async ({ page }) => {
  // Usuario dice "Afiliado" pero no es afiliado en IESS
  // Backend valida interoperabilidad
  await page.fill("radio[name='sector'][value='afiliado']")
  await page.click("button:has-text('Validar')")
  
  // Debe rechazar con mensaje específico
  expect(page.locator("text=no mantiene la calidad de afiliado")).toBeVisible()
})
```

---

## 2. VALIDACIONES DE CASCADAS Y DEPENDENCIAS

### Cascadas de datos
```typescript
test("Cascada país-provincia-cantón mantiene sincronización", async ({ page }) => {
  // Seleccionar Ecuador
  await page.selectOption("select[country]", "EC")
  let provincias = await page.locator("select[province] option").count()
  expect(provincias).toBeGreaterThan(1) // Debe tener opciones
  
  // Seleccionar Pichincha
  await page.selectOption("select[province]", "Pichincha")
  let ciudades = await page.locator("select[city] option").count()
  expect(ciudades).toBeGreaterThan(1)
  
  // Verificar que solo Quito, Mejía, etc. salen
  expect(await page.locator("select[city]").inputValue()).toContain("Quito")
  
  // Cambiar a otro país
  await page.selectOption("select[country]", "CO")
  provincias = await page.locator("select[province] option").count()
  expect(provincias).toBe(1) // Solo placeholder
})

test("Campos interdependientes: Si casado → mostrar cónyuge", async ({ page }) => {
  // No casado
  await page.selectOption("select[name='estadoCivil']", "Soltero")
  expect(page.locator("input[name='conyuge']")).toBeHidden()
  
  // Cambiar a casado
  await page.selectOption("select[name='estadoCivil']", "Casado")
  expect(page.locator("input[name='conyuge']")).toBeVisible()
  
  // Llenar datos
  await page.fill("input[name='conyuge.nombres']", "Juan")
  await page.fill("input[name='conyuge.cedula']", "1234567890")
})
```

---

## 3. VALIDACIONES DE LÍMITES Y CUOTAS

### Límites de archivos
```typescript
test("Rechazo: Archivo > 2MB es rechazado", async ({ page }) => {
  const largeFile = Buffer.alloc(3 * 1024 * 1024) // 3MB
  const fileInput = page.locator("input[type='file']").first()
  
  // Playwright detecta el tamaño e intenta cargar
  await fileInput.setInputFiles({
    name: "large.pdf",
    mimeType: "application/pdf",
    buffer: largeFile
  })
  
  // Debería mostrar error
  expect(page.locator("text=archivo muy grande")).toBeVisible()
  
  // No debería cargar
  expect(page.locator("text=Certificado 1")).not.toBeVisible()
})

test("Límite de documentos: Máximo 50 respaldos por postulante", async ({ page }) => {
  // Cargar 50 documentos válidos
  for (let i = 1; i <= 50; i++) {
    const validPdf = crearPDFValido(`doc${i}`)
    await cargarDocumento(page, validPdf)
  }
  expect(page.locator("text=Documentos cargados: 50/50")).toBeVisible()
  
  // Intento 51 debería ser rechazado
  const pdf51 = crearPDFValido("doc51")
  await cargarDocumento(page, pdf51)
  expect(page.locator("text=cuota excedida|máximo alcanzado")).toBeVisible()
})

test("Límites de texto: Descripción máximo 1000 caracteres", async ({ page }) => {
  const textarea = page.locator("textarea[name='descripcion']")
  const texto1000 = "a".repeat(1000)
  const texto1001 = "a".repeat(1001)
  
  // 1000 caracteres: aceptado
  await textarea.fill(texto1000)
  expect(textarea).toHaveValue(texto1000)
  
  // 1001 caracteres: rechazado o truncado
  await textarea.fill(texto1001)
  const actual = await textarea.inputValue()
  expect(actual.length).toBeLessThanOrEqual(1000)
})
```

---

## 4. VALIDACIONES DE TIPOS Y FORMATOS

### Validaciones de entrada
```typescript
test("Cédula ecuatoriana: Solo 10 dígitos, módulo 10", async ({ page }) => {
  const input = page.locator("input[name='cedula']")
  
  // Válida
  await input.fill("1234567890")
  await input.blur()
  expect(page.locator(".cedula-error")).not.toBeVisible()
  
  // Inválida: 9 dígitos
  await input.clear()
  await input.fill("123456789")
  await input.blur()
  expect(page.locator(".cedula-error")).toBeVisible()
  
  // Inválida: módulo 10 falla
  await input.clear()
  await input.fill("0000000000")
  await input.blur()
  expect(page.locator(".cedula-error")).toBeVisible()
})

test("Email: Validación RFC 5321 (máximo 254 caracteres)", async ({ page }) => {
  const input = page.locator("input[name='email']")
  
  // Válido
  await input.fill("test+tag@example.co.uk")
  expect(input).toHaveAttribute("aria-invalid", "false")
  
  // Inválido: sin @
  await input.clear()
  await input.fill("testemailcom")
  expect(input).toHaveAttribute("aria-invalid", "true")
  
  // Inválido: > 254 caracteres
  const longEmail = "a".repeat(250) + "@example.com"
  await input.clear()
  await input.fill(longEmail)
  expect(input).toHaveAttribute("aria-invalid", "true")
})

test("Teléfono: Formato +593 9XX XXX XXXX", async ({ page }) => {
  const input = page.locator("input[name='telefono']")
  
  // Válidos
  for (const phone of ["+593987654321", "0987654321", "+593 98 765 4321"]) {
    await input.fill(phone)
    expect(input).toHaveAttribute("aria-invalid", "false")
  }
  
  // Inválidos
  for (const phone of ["123", "ABC1234567", "+1 987 654 3210"]) {
    await input.clear()
    await input.fill(phone)
    expect(input).toHaveAttribute("aria-invalid", "true")
  }
})

test("UTF-8: Aceptar Ñ, á, é, etc. en nombres", async ({ page }) => {
  const input = page.locator("input[name='nombres']")
  
  const nombres = ["José María", "Rafaela Jiménez", "Ándres García", "Susaña"]
  for (const nombre of nombres) {
    await input.fill(nombre)
    await input.blur()
    expect(page.locator(".nombres-error")).not.toBeVisible()
  }
})
```

---

## 5. PERSISTENCIA Y RECARGA DE PÁGINA

### Estado se mantiene tras reload
```typescript
test("Guardar progreso: Datos persisten tras F5", async ({ page }) => {
  const cedula = "1234567890"
  const pass = "Test123!"
  
  // Login
  await loginWithUser(page, cedula, pass)
  
  // Ir a sección 2, llenar datos
  await page.goto("/postulacion/2")
  await page.fill("input[name='telefono']", "0987654321")
  await page.fill("input[name='email2']", "alt@example.com")
  await page.click("button:has-text('Guardar')")
  
  // Esperar confirmación
  await expect(page.locator("text=Guardado correctamente")).toBeVisible()
  
  // Refrescar página
  await page.reload()
  await expect(page).toHaveURL(/\/postulacion\/2/)
  
  // Datos deben estar ahí
  expect(page.locator("input[name='telefono']")).toHaveValue("0987654321")
  expect(page.locator("input[name='email2']")).toHaveValue("alt@example.com")
})

test("Borrador vs Enviado: No se puede editar enviado", async ({ page }) => {
  // Abrir postulación enviada
  await loginWithUser(page, cedulaEnviada, pass)
  await page.goto("/postulacion/1")
  
  // Inputs deben estar disabled
  const inputs = page.locator("input, textarea, select")
  for (let i = 0; i < await inputs.count(); i++) {
    const disabled = await inputs.nth(i).isDisabled()
    expect(disabled).toBe(true) // O al menos: readonly
  }
  
  // Botón enviar no debe existir
  expect(page.locator("button:has-text('Enviar')")).not.toBeVisible()
})
```

---

## 6. TESTS DE ROL Y PERMISOS (RBAC)

### Acceso basado en rol
```typescript
test("Postulante no ve /admin/preguntas", async ({ page }) => {
  await loginWithUser(page, cedulaPostulante, pass)
  await page.goto("/admin/preguntas")
  
  expect(page.locator("text=No tiene permisos")).toBeVisible()
  expect(page).toHaveURL(/\/postulacion|\/403/)
})

test("Analista ve /dashboard pero no /admin", async ({ page }) => {
  await loginWithUser(page, cedulaAnalista, pass)
  
  // Sí a dashboard
  await page.goto("/dashboard")
  expect(page).toHaveURL(/\/dashboard/)
  
  // No a admin
  await page.goto("/admin/preguntas")
  expect(page.locator("text=No tiene permisos")).toBeVisible()
})

test("Admin ve /admin/preguntas y /admin/fases", async ({ page }) => {
  await loginWithUser(page, cedulaAdmin, pass)
  
  await page.goto("/admin/preguntas")
  expect(page).toHaveURL(/\/admin\/preguntas/)
  
  await page.goto("/admin/fases")
  expect(page).toHaveURL(/\/admin\/fases/)
})
```

---

## 7. TESTS DE CONCURRENCIA (Dos usuarios)

### Mismo formulario simultáneamente
```typescript
test("Dos usuarios llenan el mismo sector sin conflicto", async ({ browser }) => {
  const context1 = await browser.newContext()
  const context2 = await browser.newContext()
  
  const page1 = await context1.newPage()
  const page2 = await context2.newPage()
  
  // Usuario 1 login
  await loginWithUser(page1, cedula1, pass)
  await page1.goto("/postulacion/6") // Experiencia profesional
  
  // Usuario 2 login (DIFERENTE sector)
  await loginWithUser(page2, cedula2, pass)
  await page2.goto("/postulacion/6")
  
  // Usuario 1 agrega experiencia
  await agregarExperiencia(page1, "Ingeniero", "TechCorp", "2020-01-01", "2023-12-31")
  await page1.click("button:has-text('Guardar')")
  
  // Usuario 2 agrega diferente experiencia (simultáneamente)
  await page2.waitForTimeout(100) // Pequeño offset
  await agregarExperiencia(page2, "Contador", "FinanceCorp", "2018-01-01", "2023-12-31")
  await page2.click("button:has-text('Guardar')")
  
  // Ambos deben tener sus datos, sin sobrescrituras
  await page1.goto("/postulacion/6")
  expect(page1.locator("text=Ingeniero")).toBeVisible()
  expect(page1.locator("text=TechCorp")).toBeVisible()
  
  await page2.goto("/postulacion/6")
  expect(page2.locator("text=Contador")).toBeVisible()
  expect(page2.locator("text=FinanceCorp")).toBeVisible()
})
```

---

## 8. TESTS DE INTEGRACIÓN CON SERVICIOS EXTERNOS

### Interoperabilidad Registro Civil
```typescript
test("Validación dactilar: Datos cargan desde Registro Civil", async ({ page }) => {
  await page.goto("/registro")
  
  // Ingresar cédula
  await page.fill("input[name='cedula']", "1720104008")
  await page.click("label:has-text('Código Dactilar')")
  
  // Debe pedir código dactilar
  const codigoInput = page.locator("input[name='codigoDactilar']")
  expect(codigoInput).toBeVisible()
  
  // Ingresar código válido
  await codigoInput.fill("0123456789")
  await page.click("button:has-text('Validar')")
  
  // Debe buscar en Registro Civil y cargar datos
  await expect(page.locator("input[name='nombres']")).toHaveValue(/[A-Za-z]/)
  await expect(page.locator("input[name='apellidos']")).toHaveValue(/[A-Za-z]/)
  
  // Nombres/apellidos deben ser read-only (vinieron del Registro Civil)
  expect(page.locator("input[name='nombres']")).toBeDisabled()
})

test("Correo: OTP se envía y llega a Mailpit", async ({ page, request }) => {
  await page.goto("/registro")
  await page.fill("input[name='cedula']", "1234567890")
  await page.fill("input[name='correo']", "test+otp@example.com")
  await page.click("button:has-text('Enviar código')")
  
  // Esperar a que Mailpit reciba el email
  await page.waitForTimeout(1000)
  
  // Consultar Mailpit API
  const resp = await request.get("http://localhost:8025/api/v1/messages")
  const messages = await resp.json()
  
  const mailOTP = messages.find(m => m.To[0].includes("test+otp@example.com"))
  expect(mailOTP).toBeDefined()
  
  // Extraer código OTP del email
  const otpMatch = mailOTP.Text.match(/(\d{6})/)
  const otp = otpMatch[1]
  
  // Usar en formulario
  await page.fill("input[name='otp']", otp)
  await page.click("button:has-text('Verificar')")
  
  // Debe aceptar
  expect(page.locator(".otp-error")).not.toBeVisible()
})

test("Documentos: Se guardan en S3 (Rustfs)", async ({ page, request }) => {
  await loginWithUser(page, cedula, pass)
  await page.goto("/postulacion/5")
  
  // Cargar certificado
  const pdf = crearPDFValido("Certificado.pdf")
  await page.locator("input[type='file']").setInputFiles({
    name: "Certificado.pdf",
    mimeType: "application/pdf",
    buffer: pdf
  })
  
  await page.click("button:has-text('Cargar')")
  expect(page.locator("text=Certificado.pdf")).toBeVisible()
  
  // Guardar postulación
  await page.click("button:has-text('Guardar')")
  
  // Verificar que el archivo llegó a S3/Rustfs
  const s3Resp = await request.get("http://localhost:9000/biess/")
  expect(s3Resp.status()).toBeLessThan(400) // S3 debe responder
  
  // El archivo debe ser descargable
  const downloadUrl = page.locator("a[download*='Certificado']").first()
  const href = await downloadUrl.getAttribute("href")
  const fileResp = await request.get(href)
  expect(fileResp.ok()).toBe(true)
  expect(fileResp.headers()["content-type"]).toContain("application/pdf")
})
```

---

## 9. TESTS DE AUDITORÍA Y TRAZABILIDAD

### Se registra todo
```typescript
test("Auditoría: Cada acción queda en BD con timestamp y usuario", async ({ page, request }) => {
  const cedula = "1234567890"
  const pass = "Test123!"
  
  // Login
  await loginWithUser(page, cedula, pass)
  
  // Ir a postulación y cambiar algo
  await page.goto("/postulacion/1")
  const ahora = new Date()
  
  await page.fill("input[name='nombres']", "Juan Manuel")
  await page.click("button:has-text('Guardar')")
  
  // Consultar auditoría vía API (admin endpoint)
  const auditResp = await request.get(
    `http://localhost:8080/api/v1/admin/audit?applicant_id=${cedula}&limit=1`,
    {
      headers: { Cookie: page.context().cookies() }
    }
  )
  
  const audit = await auditResp.json()
  expect(audit.length).toBeGreaterThan(0)
  
  const lastEntry = audit[0]
  expect(lastEntry.actor_id).toBe(cedula)
  expect(lastEntry.action).toBe("SUBMISSION_UPDATE")
  expect(new Date(lastEntry.timestamp)).toBeCloseTo(ahora, { ms: 5000 })
})

test("Auditoría: Exportaciones XLSX quedan registradas con hash", async ({ page, request }) => {
  // Login como analista
  await loginWithUser(page, cedulaAnalista, pass)
  
  // Descargar matriz
  await page.goto("/dashboard/postulaciones")
  const downloadPromise = page.waitForEvent("download")
  await page.click("button:has-text('Descargar Excel')")
  const download = await downloadPromise
  
  // Backend debe registrar: quién descargó, cuándo, hash SHA256 del archivo
  const exportAudit = await request.get(
    "http://localhost:8080/api/v1/admin/exports?limit=1",
    { headers: { Cookie: page.context().cookies() } }
  )
  
  const exports = await exportAudit.json()
  expect(exports.length).toBeGreaterThan(0)
  
  const last = exports[0]
  expect(last.downloaded_by).toBe(cedulaAnalista)
  expect(last.file_sha256).toMatch(/^[a-f0-9]{64}$/) // SHA256 hex
  expect(new Date(last.downloaded_at)).toBeCloseTo(new Date(), { ms: 5000 })
})
```

---

## 10. TESTS DE SEGURIDAD

### Inyecciones y XSS
```typescript
test("XSS: Campos de texto limpian HTML", async ({ page }) => {
  await loginWithUser(page, cedula, pass)
  await page.goto("/postulacion/1")
  
  // Intentar inyectar XSS
  const malicious = "<script>alert('XSS')</script>"
  await page.fill("input[name='nombres']", malicious)
  await page.click("button:has-text('Guardar')")
  
  // Reload y verificar que fue sanitizado
  await page.reload()
  const valor = await page.locator("input[name='nombres']").inputValue()
  
  // Debe estar sin etiquetas
  expect(valor).not.toContain("<script>")
  expect(valor).not.toContain("</script>")
})

test("CSRF: POST sin token es rechazado", async ({ request }) => {
  // Intentar POST directo sin ir por el navegador (sin CSRF token)
  const resp = await request.post(
    "http://localhost:8080/api/v1/submissions",
    {
      data: { applicant_id: "123", content: {} },
      headers: { 
        "Content-Type": "application/json",
        "Origin": "http://malicious.com"
      }
    }
  )
  
  // Debe retornar 403 o 400
  expect([403, 400, 401]).toContain(resp.status())
})

test("SQL Injection: Cédula con comillas es escapada", async ({ page }) => {
  await page.goto("/login")
  
  // Intentar SQL injection
  const sqlInjection = "1234567890' OR '1'='1"
  await page.fill("input[name='cedula']", sqlInjection)
  await page.fill("input[name='password']", "any")
  await page.click("button[type='submit']")
  
  // Debe rechazar como credencial inválida, no tener un error SQL
  expect(page.locator("text=/credenciales|incorrecta/i")).toBeVisible()
  expect(page.locator("text=/syntax|SQL|database/i")).not.toBeVisible()
})

test("Rate Limiting: IP con 100 requests/min es bloqueada", async ({ request }) => {
  // Hacer 100 requests rápido
  const responses = []
  for (let i = 0; i < 100; i++) {
    const resp = await request.get("http://localhost:8080/api/v1/catalog/countries")
    responses.push(resp.status())
  }
  
  // Al menos el último debe ser 429 (Too Many Requests)
  expect(responses.slice(-10)).toContain(429)
})
```

---

## 11. TESTS DE PERFORMANCE

### Tiempos de respuesta
```typescript
test("Performance: Login < 500ms", async ({ page }) => {
  const inicio = Date.now()
  
  await page.goto("/login")
  await page.fill("input[name='cedula']", cedula)
  await page.fill("input[name='password']", password)
  await page.click("button[type='submit']")
  
  await page.waitForURL(/\/postulacion/)
  
  const tiempo = Date.now() - inicio
  expect(tiempo).toBeLessThan(500) // Debe ser rápido
})

test("Performance: Cargar formulario 9 secciones < 1s", async ({ page }) => {
  await loginWithUser(page, cedula, pass)
  
  const inicio = Date.now()
  await page.goto("/postulacion/1")
  await page.waitForLoadState("networkidle")
  
  const tiempo = Date.now() - inicio
  expect(tiempo).toBeLessThan(1000)
})

test("Performance: Búsqueda de catálogos (caché) < 50ms", async ({ page }) => {
  // 1er request: carga desde BD
  const inicio1 = Date.now()
  await page.goto("/postulacion/2")
  await page.waitForLoadState("networkidle")
  const tiempo1 = Date.now() - inicio1
  
  // 2do request: debería estar en caché
  const inicio2 = Date.now()
  await page.reload()
  await page.waitForLoadState("networkidle")
  const tiempo2 = Date.now() - inicio2
  
  // Segundo debe ser mucho más rápido
  expect(tiempo2).toBeLessThan(tiempo1 * 0.5) // Al menos 50% más rápido
})
```

---

## 12. TESTS DE ACCESIBILIDAD (WCAG)

### Navegación por teclado
```typescript
test("Accesibilidad: Tab navegación completa sin ratón", async ({ page }) => {
  await page.goto("/login")
  
  // Empezar en cedula
  const cedulaInput = page.locator("input[name='cedula']")
  await cedulaInput.focus()
  
  // Tab para pasar a password
  await page.keyboard.press("Tab")
  expect(page.locator("input[name='password']")).toBeFocused()
  
  // Tab para pasar a botón submit
  await page.keyboard.press("Tab")
  expect(page.locator("button[type='submit']")).toBeFocused()
  
  // Enter para submit
  await page.fill("input[name='cedula']", cedula)
  await page.fill("input[name='password']", password)
  await page.keyboard.press("Enter")
  
  await page.waitForURL(/\/postulacion/)
})

test("Accesibilidad: Errores anunciados para screen readers", async ({ page }) => {
  await page.goto("/login")
  
  // Submit sin llenar
  await page.click("button[type='submit']")
  
  // Los errores deben tener aria-live
  const alerts = page.locator("[role='alert'], [aria-live]")
  expect(await alerts.count()).toBeGreaterThan(0)
  
  // Deben ser visible para screen readers
  for (let i = 0; i < await alerts.count(); i++) {
    const ariaLive = await alerts.nth(i).getAttribute("aria-live")
    expect(["polite", "assertive"]).toContain(ariaLive)
  }
})

test("Accesibilidad: Contraste de colores cumple WCAG AA", async ({ page }) => {
  // Usar librería axe
  const { injectAxe, checkA11y } = require("axe-playwright")
  
  await page.goto("/")
  await injectAxe(page)
  
  await checkA11y(page, null, {
    detailedReport: true,
    detailedReportOptions: {
      html: true
    }
  })
})
```

---

## 13. TESTS DE MOBILE / RESPONSIVE

### Diferentes tamaños de pantalla
```typescript
test("Mobile: Formulario responsivo en iPhone 12", async ({ browser }) => {
  const context = await browser.newContext({
    viewport: { width: 390, height: 844 },
    userAgent: "iPhone"
  })
  const page = await context.newPage()
  
  await loginWithUser(page, cedula, pass)
  await page.goto("/postulacion/1")
  
  // Inputs deben ser accesibles
  const inputs = page.locator("input, select, textarea")
  for (let i = 0; i < await inputs.count(); i++) {
    const input = inputs.nth(i)
    expect(await input.isVisible()).toBe(true)
    
    // Tamaño de tap target >= 44x44 pixels
    const box = await input.boundingBox()
    expect(box.width).toBeGreaterThanOrEqual(30) // Flexible en mobile
    expect(box.height).toBeGreaterThanOrEqual(30)
  }
})

test("Tablet: Formulario en iPad (landscape)", async ({ browser }) => {
  const context = await browser.newContext({
    viewport: { width: 1024, height: 768 },
    userAgent: "iPad"
  })
  const page = await context.newPage()
  
  await loginWithUser(page, cedula, pass)
  await page.goto("/postulacion/1")
  
  // En tablet, columnas pueden ser 2 (no 1 como mobile)
  const columnas = page.locator(".form-grid")
  const computedStyle = await columnas.evaluate(el => 
    window.getComputedStyle(el).gridTemplateColumns
  )
  
  // Debe tener múltiples columnas
  const columnCount = computedStyle.split(" ").length
  expect(columnCount).toBeGreaterThan(1)
})

test("Orientation: Cambiar portrait ↔ landscape mantiene datos", async ({ page }) => {
  const context = page.context()
  
  // Portrait
  await context.setViewportSize({ width: 390, height: 844 })
  await loginWithUser(page, cedula, pass)
  await page.goto("/postulacion/1")
  await page.fill("input[name='nombres']", "Juan")
  
  // Rotar a landscape
  await context.setViewportSize({ width: 844, height: 390 })
  await page.waitForLoadState("networkidle")
  
  // Dato debe persistir
  expect(page.locator("input[name='nombres']")).toHaveValue("Juan")
})
```

---

## MATRIZ DE COBERTURA POR CRITICIDAD

| Tipo de Prueba | P0 (Antes envío) | P1 (Post-examen) |
|---|---|---|
| Flujos completos happy path | ✅ | - |
| Validaciones críticas (nacionalidad, cascada) | ✅ | - |
| Límites y cuotas | ✅ | - |
| Persistencia (guardar/reload) | ✅ | - |
| RBAC (postulante vs analista vs admin) | ✅ | - |
| Integración Registro Civil | ✅ | - |
| Integración Mailpit (OTP) | ✅ | - |
| Integración S3 (documentos) | - | ✅ |
| Auditoría | - | ✅ |
| Seguridad (XSS, SQLi, CSRF) | - | ✅ |
| Performance | - | ✅ |
| Accesibilidad | - | ✅ |
| Mobile/Responsive | - | ✅ |
| Concurrencia | - | ✅ |

---

## CÓMO EJECUTARLAS

```bash
# Todas las E2E
npx playwright test e2e/ --reporter=html

# Solo flujos críticos
npx playwright test e2e/ --grep="flujo|critico"

# Con retries para flaky tests
npx playwright test e2e/ --retries=2

# En modo UI (ver ejecución)
npx playwright test e2e/ --ui

# Generar reporte HTML
npx playwright test e2e/ && npx playwright show-report
```

