import { PrismaClient } from '@prisma/client';

const prisma = new PrismaClient();

async function main() {
  console.log('🌱 Starting seed...');

  // Criar usuário
  const user = await prisma.user.upsert({
    where: { email: 'bruno@wbdigitalsolutions.com' },
    update: {},
    create: {
      email: 'bruno@wbdigitalsolutions.com',
      passwordHash: 'temp_password_hash', // Temporário, será substituído quando implementar autenticação
      name: 'Bruno Vieira',
      timezone: 'America/Sao_Paulo',
    },
  });

  console.log('✅ User created:', user.email);

  // Criar calendário WB Digital Solutions
  const calendarWB = await prisma.calendar.upsert({
    where: { id: 'wb-digital-calendar' },
    update: {},
    create: {
      id: 'wb-digital-calendar',
      userId: user.id,
      name: 'WB Digital Solutions',
      email: 'bruno@wbdigitalsolutions.com',
      color: '#350545',
      type: 'professional',
    },
  });

  console.log('✅ Calendar created:', calendarWB.name);

  // Criar calendário Bruno - Pessoal
  const calendarPersonal = await prisma.calendar.upsert({
    where: { id: 'bruno-personal-calendar' },
    update: {},
    create: {
      id: 'bruno-personal-calendar',
      userId: user.id,
      name: 'Bruno - Pessoal',
      email: 'wbrunovieira77@gmail.com',
      color: '#792990',
      type: 'personal',
    },
  });

  console.log('✅ Calendar created:', calendarPersonal.name);

  // Criar categorias para WB Digital Solutions
  const categoriesWB = await Promise.all([
    prisma.category.upsert({
      where: { id: 'cat-wb-reunioes' },
      update: {},
      create: {
        id: 'cat-wb-reunioes',
        calendarId: calendarWB.id,
        name: 'Reuniões',
        icon: '💼',
        color: '#FF5733',
        type: 'work',
      },
    }),
    prisma.category.upsert({
      where: { id: 'cat-wb-prospeccao' },
      update: {},
      create: {
        id: 'cat-wb-prospeccao',
        calendarId: calendarWB.id,
        name: 'Prospecção',
        icon: '📞',
        color: '#33A1FF',
        type: 'work',
      },
    }),
    prisma.category.upsert({
      where: { id: 'cat-wb-programacao' },
      update: {},
      create: {
        id: 'cat-wb-programacao',
        calendarId: calendarWB.id,
        name: 'Programação',
        icon: '💻',
        color: '#33FF57',
        type: 'work',
      },
    }),
    prisma.category.upsert({
      where: { id: 'cat-wb-financeiro' },
      update: {},
      create: {
        id: 'cat-wb-financeiro',
        calendarId: calendarWB.id,
        name: 'Financeiro',
        icon: '💰',
        color: '#FFD700',
        type: 'work',
      },
    }),
  ]);

  console.log(
    `✅ ${categoriesWB.length} categories created for WB Digital Solutions`,
  );

  // Criar categorias para Bruno - Pessoal
  const categoriesPersonal = await Promise.all([
    prisma.category.upsert({
      where: { id: 'cat-per-academia' },
      update: {},
      create: {
        id: 'cat-per-academia',
        calendarId: calendarPersonal.id,
        name: 'Academia',
        icon: '🏃',
        color: '#FF6B6B',
        type: 'health',
      },
    }),
    prisma.category.upsert({
      where: { id: 'cat-per-jiujitsu' },
      update: {},
      create: {
        id: 'cat-per-jiujitsu',
        calendarId: calendarPersonal.id,
        name: 'Jiu-Jitsu',
        icon: '🥋',
        color: '#4ECDC4',
        type: 'health',
      },
    }),
    prisma.category.upsert({
      where: { id: 'cat-per-natacao' },
      update: {},
      create: {
        id: 'cat-per-natacao',
        calendarId: calendarPersonal.id,
        name: 'Natação',
        icon: '🏊',
        color: '#1A535C',
        type: 'health',
      },
    }),
    prisma.category.upsert({
      where: { id: 'cat-per-meditacao' },
      update: {},
      create: {
        id: 'cat-per-meditacao',
        calendarId: calendarPersonal.id,
        name: 'Meditação',
        icon: '🧘',
        color: '#9B59B6',
        type: 'wellness',
      },
    }),
    prisma.category.upsert({
      where: { id: 'cat-per-leitura' },
      update: {},
      create: {
        id: 'cat-per-leitura',
        calendarId: calendarPersonal.id,
        name: 'Leitura',
        icon: '📖',
        color: '#3498DB',
        type: 'leisure',
      },
    }),
  ]);

  console.log(
    `✅ ${categoriesPersonal.length} categories created for Bruno - Pessoal`,
  );

  console.log('🎉 Seed completed successfully!');
}

main()
  .catch((e) => {
    console.error('❌ Error seeding database:', e);
    process.exit(1);
  })
  .finally(async () => {
    await prisma.$disconnect();
  });
