import { test, expect } from '@playwright/test';

test('Test E2E de Auth en Vercel: Login, Logout y Re-Login Inmediato', async ({ page }) => {
  test.setTimeout(90000);

  console.log('1. Navegando al Login en Vercel...');
  await page.goto('https://framework-examen.vercel.app/login');

  const cedulaInput = page.locator('input[name="cedula"], input[id*="cedula"], input[type="text"]').first();
  const passwordInput = page.locator('input[type="password"]').first();
  const submitBtn = page.getByRole('button', { name: /iniciar sesión|ingresar|acceder/i });

  console.log('2. Primer Login con credenciales...');
  await cedulaInput.fill('1710000017');
  await passwordInput.fill('Password123!');
  await submitBtn.click();

  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 20000 });
  console.log('URL tras primer login:', page.url());
  expect(page.url()).toContain('/postulacion');

  // Esperar a que la cabecera cargue el nombre del usuario
  await expect(page.getByText('Yandris Rivera')).toBeVisible({ timeout: 10000 });

  console.log('3. Haciendo Logout (Cerrar sesión)...');
  const salirBtn = page.getByRole('button', { name: 'Salir' });
  await expect(salirBtn).toBeVisible();
  
  await Promise.all([
    page.waitForResponse((resp) => resp.url().includes('/auth/logout'), { timeout: 15000 }).catch(() => null),
    salirBtn.click(),
  ]);

  await page.waitForURL((url) => url.pathname.includes('/login'), { timeout: 15000 });
  console.log('URL tras logout:', page.url());
  expect(page.url()).toContain('/login');

  console.log('4. Intentando RE-LOGIN inmediato...');
  await page.waitForSelector('input[name="cedula"], input[id*="cedula"], input[type="text"]');
  const cedulaInput2 = page.locator('input[name="cedula"], input[id*="cedula"], input[type="text"]').first();
  const passwordInput2 = page.locator('input[type="password"]').first();
  const submitBtn2 = page.getByRole('button', { name: /iniciar sesión|ingresar|acceder/i });

  await cedulaInput2.fill('1710000017');
  await passwordInput2.fill('Password123!');
  await submitBtn2.click();

  console.log('5. Verificando resultado del re-login...');
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 20000 });
  console.log('URL tras re-login exitoso:', page.url());
  expect(page.url()).toContain('/postulacion');

  await expect(page.getByText('Yandris Rivera')).toBeVisible({ timeout: 10000 });

  console.log('¡Re-login completado sin errores!');
});
