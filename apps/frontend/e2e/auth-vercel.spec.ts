import { test, expect } from '@playwright/test';

test('Test E2E de Auth en Vercel: Login, Validacion de Sesion y Logout', async ({ page }) => {
  test.setTimeout(90000);

  console.log('1. Navegando a la pantalla de Login en Vercel...');
  await page.goto('https://framework-examen.vercel.app/login');

  await page.waitForSelector('input[name="cedula"], input[id*="cedula"], input[type="text"]');
  
  console.log('2. Ingresando credenciales del postulante registrado...');
  const cedulaInput = page.locator('input[name="cedula"], input[id*="cedula"]').first();
  const passwordInput = page.locator('input[type="password"]').first();

  await cedulaInput.fill('1710000017');
  await passwordInput.fill('Password123!');

  console.log('3. Haciendo clic en Iniciar Sesión...');
  const submitBtn = page.getByRole('button', { name: /iniciar sesión|ingresar|acceder/i });
  await submitBtn.click();

  console.log('4. Esperando respuesta de autenticación y redirección a /postulacion...');
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 20000 });
  
  console.log('URL tras iniciar sesión:', page.url());
  expect(page.url()).toContain('/postulacion');

  console.log('5. Verificando que la interfaz muestre el nombre del postulante (Yandris Rivera)...');
  await page.waitForTimeout(2000);
  const bodyText = await page.textContent('body');
  expect(bodyText).toContain('Yandris');

  console.log('6. Haciendo clic en el botón Salir...');
  const salirBtn = page.getByRole('button', { name: 'Salir' });
  await expect(salirBtn).toBeVisible();
  
  // Esperar a que la petición de logout y la navegación concluyan
  await Promise.all([
    page.waitForResponse((resp) => resp.url().includes('/auth/logout'), { timeout: 15000 }).catch(() => null),
    salirBtn.click(),
  ]);

  console.log('7. Verificando que tras cerrar sesión estemos en login...');
  await page.waitForTimeout(2000);
  console.log('URL tras cerrar sesión:', page.url());
  expect(page.url()).toContain('/login');

  console.log('8. Limpiando contexto de cookies o recargando para verificar expiración de sesión...');
  await page.context().clearCookies();
  await page.goto('https://framework-examen.vercel.app/postulacion');
  await page.waitForURL((url) => url.pathname.includes('/login'), { timeout: 20000 });
  console.log('URL tras reintento sin cookies:', page.url());
  expect(page.url()).toContain('/login');

  console.log('¡Prueba E2E de Autenticación completada con éxito rotundo!');
});
