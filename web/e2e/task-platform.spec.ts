import { expect, type Locator, type Page, test } from '@playwright/test'

async function register(page: Page) {
  await page.goto('/register')
  await page.getByLabel('用户名').fill('phase10_user')
  await page.getByLabel('邮箱').fill('phase10@example.com')
  await page.getByLabel('密码').fill('password123')
  await page.getByRole('button', { name: /注\s*册/ }).click()
  await expect(page.getByRole('heading', { name: '我的项目' })).toBeVisible()
}

async function resetSession(page: Page) {
  await page.evaluate(() => window.localStorage.removeItem('task-platform:access-token'))
  await page.goto('/login')
  await expect(page.getByRole('heading', { name: '登录' })).toBeVisible()
}

async function login(page: Page, account = 'demo_user') {
  await page.goto('/login')
  await page.getByLabel('账号或邮箱').fill(account)
  await page.getByLabel('密码').fill('password123')
  await page.getByRole('button', { name: /登\s*录/ }).click()
  await expect(page.getByRole('heading', { name: '我的项目' })).toBeVisible()
}

async function loginAndOpenProject(page: Page, account: string) {
  await login(page, account)
  await page.goto('/projects/project-web-console?tab=settings')
  await expect(page.getByRole('heading', { name: '前端管理台' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '成员管理' })).toBeVisible()
}

function rowForMember(page: Page, userId: string) {
  return page.locator('tr').filter({ hasText: userId })
}

async function createProject(page: Page, projectName: string) {
  await page.getByRole('button', { name: '创建项目' }).first().click()
  const dialog = page.getByRole('dialog', { name: '创建项目' })
  await dialog.getByLabel('项目名称').fill(projectName)
  await dialog.getByLabel('项目描述').fill('阶段10 E2E 项目')
  await dialog.getByRole('button', { name: /创\s*建/ }).click()
  await expect(page.getByText(projectName)).toBeVisible()
  await page.getByRole('link', { name: '进入项目' }).first().click()
  await expect(page.getByRole('heading', { name: projectName })).toBeVisible()
}

async function addMember(page: Page, userId: string) {
  await page.getByRole('tab', { name: '项目设置' }).click()
  await page.getByRole('button', { name: '添加成员' }).click()
  const dialog = page.getByRole('dialog', { name: '添加成员' })
  await dialog.getByLabel('用户 ID').fill(userId)
  await dialog.getByRole('button', { name: /添\s*加/ }).click()
  await expect(rowForMember(page, userId)).toBeVisible()
}

async function createTask(page: Page, taskTitle: string) {
  await page.getByRole('tab', { name: '看板' }).click()
  await page.getByRole('button', { name: '创建任务' }).click()
  const dialog = page.getByRole('dialog', { name: '创建任务' })
  await dialog.getByLabel('任务标题').fill(taskTitle)
  await dialog.getByLabel('任务内容').fill('拖拽、评论、归档验收')
  await dialog.getByRole('button', { name: /创\s*建/ }).click()
  await expect(page.getByText(taskTitle)).toBeVisible()
}

async function dragTaskToDoing(page: Page, taskTitle: string) {
  const taskCard = page.locator('.task-card').filter({ hasText: taskTitle })
  const doingColumn = page.getByRole('region', { name: '进行中' })
  await taskCard.dragTo(doingColumn)
  await expect(doingColumn.getByText(taskTitle)).toBeVisible()
}

async function commentOnTask(page: Page, taskTitle: string, comment: string) {
  await page.getByRole('button', { name: taskTitle }).click()
  const drawer = page.getByRole('dialog', { name: taskTitle })
  await drawer.getByLabel('评论内容').fill(comment)
  await drawer.getByRole('button', { name: '发送' }).click()
  await expect(drawer.getByText(comment)).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(drawer).not.toBeVisible()
}

async function archiveProject(page: Page) {
  await page.getByRole('tab', { name: '项目设置' }).click()
  await page.getByRole('button', { name: '归档项目' }).click()
  await page.getByRole('button', { name: /^归\s*档$/ }).click()
  await expect(page.getByText('已归档').first()).toBeVisible()
  await page.getByRole('tab', { name: '看板' }).click()
  await expect(page.getByRole('button', { name: '创建任务' })).toBeDisabled()
}

async function expectDisabled(button: Locator) {
  await expect(button).toBeDisabled()
}

test.describe('task platform E2E', () => {
  test('covers register, login, project setup, task drag, comment, and archive', async ({
    page,
  }) => {
    const projectName = `阶段10项目 ${Date.now()}`
    const taskTitle = `阶段10任务 ${Date.now()}`
    const comment = '阶段10 E2E 评论'

    await register(page)
    await resetSession(page)
    await login(page)
    await createProject(page, projectName)
    await addMember(page, 'user-4')
    await createTask(page, taskTitle)
    await dragTaskToDoing(page, taskTitle)
    await commentOnTask(page, taskTitle, comment)
    await archiveProject(page)
  })

  test('enforces project settings permission matrix', async ({ page }) => {
    await loginAndOpenProject(page, 'demo_user')
    await expect(page.getByRole('button', { name: '编辑项目' })).toBeEnabled()
    await expect(page.getByRole('button', { name: '添加成员' })).toBeEnabled()
    await expect(page.getByLabel('修改成员 user-2 角色').first()).toBeVisible()
    await expect(page.getByRole('button', { name: '归档项目' })).toBeEnabled()

    await resetSession(page)
    await loginAndOpenProject(page, 'admin_user')
    await expectDisabled(page.getByRole('button', { name: '编辑项目' }))
    await expect(page.getByRole('button', { name: '添加成员' })).toBeEnabled()
    await expect(page.getByLabel('移除成员 user-3')).toBeVisible()
    await expect(page.getByLabel('修改成员 user-3 角色')).toHaveCount(0)
    await expectDisabled(page.getByRole('button', { name: '归档项目' }))

    await resetSession(page)
    await loginAndOpenProject(page, 'member_user')
    await expectDisabled(page.getByRole('button', { name: '编辑项目' }))
    await expectDisabled(page.getByRole('button', { name: '添加成员' }))
    await expectDisabled(page.getByRole('button', { name: '归档项目' }))
    await expect(page.getByRole('button', { name: '退出项目' })).toBeEnabled()
  })
})
