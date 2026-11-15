import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { RoutePage } from './RoutePage';
import { useRouteStore } from '@/entities/trainee-route';

describe('RoutePage (1 in 1 with route.svg)', () => {
  beforeEach(() => {
    useRouteStore.getState().resetRoute();
  });

  it('renders top header and intern switcher tabs', () => {
    render(<RoutePage />);

    expect(screen.getByText('Мои стажеры')).toBeInTheDocument();
    expect(screen.getAllByText('Анна Смирнова').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('Иван Орлов')).toBeInTheDocument();
    expect(screen.queryByText('Обновить')).not.toBeInTheDocument();
  });

  it('renders goal and expectations summary cards', () => {
    render(<RoutePage />);

    expect(screen.getByText('ЦЕЛЬ')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Самостоятельно выпустить первый PR и подготовиться к экватору.',
      ),
    ).toBeInTheDocument();

    expect(screen.getByText('ОЖИДАНИЯ')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Уточнять блокеры на 1:1; фиксировать результаты в трекере.',
      ),
    ).toBeInTheDocument();
  });

  it('renders intern info, progress bar, and 6 task cards matching route.svg', () => {
    render(<RoutePage />);

    expect(screen.getByText('JUNIOR FRONTEND DEVELOPER')).toBeInTheDocument();
    expect(screen.getByText('Мария Соколова')).toBeInTheDocument();
    expect(screen.getByText('01.11.2026')).toBeInTheDocument();

    expect(screen.getByText('Прогресс адаптации')).toBeInTheDocument();
    expect(screen.getByText(/2 из 6/)).toBeInTheDocument();
    expect(screen.getByText('(33%)')).toBeInTheDocument();

    // Check all 6 tasks from route.svg
    expect(screen.getByText('Познакомься с командой')).toBeInTheDocument();
    expect(
      screen.getByText('Запроси доступ в пространство команды'),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Внеси цели, задачи и ожидания в трекер'),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Изучи документы и мануалы команды'),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Изучи памятку о закрытии практики'),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Подготовь презентацию и речь к экваторной встрече'),
    ).toBeInTheDocument();
  });

  it('switches active intern when clicking tab', () => {
    render(<RoutePage />);

    fireEvent.click(screen.getByText('Иван Орлов'));

    expect(screen.getByText('JUNIOR BACKEND DEVELOPER')).toBeInTheDocument();
    expect(screen.getByText('Сергей Волков')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Развернуть тестовый микросервис на Go и настроить дашборды Grafana.',
      ),
    ).toBeInTheDocument();
  });

  it('toggles task status and recalculates progress', () => {
    render(<RoutePage />);

    // Toggle Task 4
    const task4Btn = screen.getByLabelText(
      'Задание 4. Нажмите, чтобы отметить выполненным',
    );
    fireEvent.click(task4Btn);

    expect(screen.getByText(/3 из 6/)).toBeInTheDocument();
    expect(screen.getByText('(50%)')).toBeInTheDocument();
  });
});
