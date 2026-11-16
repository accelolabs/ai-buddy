import type {
  SSEEvent,
  TicketDraftPayload,
  TicketFlowStep,
  MentorPayload,
  CreatedTicketPayload,
} from '../model/types';
import { createTicket } from './ticketApi';

export const DEFAULT_MENTOR: MentorPayload = {
  id: 'mentor-1',
  name: 'Мария Соколова',
  role: 'Наставник стажёров',
  team: 'Команда онбординга',
  schedule: 'Отвечает в будни с 10:00 до 19:00',
  initials: 'МС',
  isOnline: true,
};

const NO_ANSWER_KEYWORDS = ['наставник', 'нет ответа', 'кто отвечает', 'зарплат'];

const NO_ANSWER_TEXT =
  'В базе знаний нет ответа на этот вопрос, а гадать я не буду. За такие вопросы отвечает ваш наставник — напишите ему, он поможет.';

const EQUATOR_TEXT =
  'Информацию о заполнении экватора вы найдёте в регламенте стажировки. Форма отчёта прикреплена внутри раздела.';

export interface StreamContext {
  flowStep?: TicketFlowStep;
  currentDraft?: TicketDraftPayload | null;
  lastUserQuery?: string;
}

function createDelay(signal: AbortSignal) {
  return (ms: number) =>
    new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(resolve, ms);
      signal.addEventListener('abort', () => {
        clearTimeout(timeout);
        reject(new DOMException('Aborted', 'AbortError'));
      });
    });
}

async function streamWords(
  text: string,
  onEvent: (event: SSEEvent) => void,
  getSeq: () => number,
  delay: (ms: number) => Promise<void>,
) {
  const words = text.split(' ');
  for (let i = 0; i < words.length; i++) {
    const chunk = (i > 0 ? ' ' : '') + words[i];
    onEvent({
      type: 'delta',
      seq: getSeq(),
      delta: chunk,
    });
    await delay(25);
  }
}

export function isTicketTrigger(text: string): boolean {
  const normalized = text.toLowerCase().trim();
  return (
    /^(?:buddy|бадди)?[,\s]*(?:создай|создать|открой|напиши|сформируй)\s*(?:тикет|обращение|заявку)/i.test(
      normalized,
    ) ||
    normalized === 'создай тикет' ||
    normalized === 'создать тикет' ||
    normalized === 'тикет' ||
    normalized === '/ticket'
  );
}

export function isApproval(text: string): boolean {
  const normalized = text
    .toLowerCase()
    .trim()
    .replace(/[.!?,]/g, '');
  const approvalKeywords = [
    'подтверждаю',
    'подтвердить',
    'подтверждаем',
    'отправляй',
    'отправить',
    'отправь',
    'все ок',
    'всё ок',
    'все верно',
    'всё верно',
    'все отлично',
    'всё отлично',
    'да',
    'ок',
    'ok',
    'хорошо',
    'согласен',
    'верно',
  ];
  return approvalKeywords.includes(normalized);
}

export function isCancellation(text: string): boolean {
  const normalized = text.toLowerCase().trim();
  return /^(?:отмена|отмени|забудь|не надо|не нужно|стоп|cancel)$/i.test(normalized);
}

export function extractSubjectAndMessage(problemText: string): {
  subject: string;
  message: string;
} {
  const trimmed = problemText.trim();

  let subject = 'Вопрос по разработке и окружению';

  const lower = trimmed.toLowerCase();
  if (lower.includes('docker') || lower.includes('докер')) {
    subject = 'Сложность при работе с Docker-контейнерами';
  } else if (lower.includes('порт') || lower.includes('port')) {
    subject = 'Конфликт портов при локальном запуске';
  } else if (lower.includes('postgres') || lower.includes('баз') || lower.includes('бд')) {
    subject = 'Проблема с подключением к базе данных';
  } else if (lower.includes('сборк') || lower.includes('build') || lower.includes('компил')) {
    subject = 'Ошибка сборки проекта';
  } else if (lower.includes('тест') || lower.includes('test')) {
    subject = 'Падение тестов при запуске';
  } else if (trimmed.length > 5 && trimmed.length <= 60) {
    subject = trimmed.charAt(0).toUpperCase() + trimmed.slice(1);
  } else {
    const firstSentence = trimmed.split(/[.!?\n]/)[0].trim();
    if (firstSentence.length > 5 && firstSentence.length <= 60) {
      subject = firstSentence.charAt(0).toUpperCase() + firstSentence.slice(1);
    } else {
      subject = trimmed.slice(0, 50).trim() + '...';
    }
  }

  const message = [
    'Здравствуйте, Мария!',
    '',
    'Столкнулся со следующей сложностью в процессе работы:',
    trimmed,
    '',
    'Подскажите, пожалуйста, как лучше решить данную проблему или к кому обратиться.',
  ].join('\n');

  return { subject, message };
}

export function refineDraft(
  currentDraft: TicketDraftPayload,
  refinementText: string,
): TicketDraftPayload {
  const lower = refinementText.toLowerCase().trim();
  let nextSubject = currentDraft.subject;
  let nextMessage = currentDraft.message;

  const subjectMatch = refinementText.match(
    /(?:измени|поменяй|укажи|сделай)\s+тему\s*(?:на|:)?\s*["«']?([^"»'\n]+)["»']?/i,
  );
  if (subjectMatch && subjectMatch[1]) {
    nextSubject = subjectMatch[1].trim();
  }

  const messageMatch = refinementText.match(
    /(?:измени|поменяй|перепиши)\s+(?:сообщение|текст|описание)\s*(?:на|:)?\s*["«']?([^"»'\n]+)["»']?/i,
  );
  if (messageMatch && messageMatch[1]) {
    nextMessage = messageMatch[1].trim();
  } else {
    const addMatch = refinementText.match(
      /(?:добавь|допиши|уточни|вставь)\s*(?:в сообщение|в текст|что|, что)?\s*[:]?\s*["«']?([^"»'\n]+)["»']?/i,
    );
    if (addMatch && addMatch[1]) {
      const addition = addMatch[1].trim();
      nextMessage = `${nextMessage}\n\nДополнение:\n${addition}`;
    } else if (!subjectMatch) {
      if (
        lower.startsWith('измени') ||
        lower.startsWith('поменяй') ||
        lower.startsWith('поправь')
      ) {
        const cleaned = refinementText
          .replace(/^(?:измени|поменяй|поправь|перепиши)\s*/i, '')
          .trim();
        if (cleaned.length > 0 && cleaned.length <= 50) {
          nextSubject = cleaned.charAt(0).toUpperCase() + cleaned.slice(1);
        } else {
          nextMessage = `${nextMessage}\n\nУточнение от стажёра:\n${cleaned}`;
        }
      } else {
        nextMessage = `${nextMessage}\n\nУточнение:\n${refinementText.trim()}`;
      }
    }
  }

  return {
    ...currentDraft,
    subject: nextSubject,
    message: nextMessage,
  };
}

export async function simulateSSEStream(
  userText: string,
  onEvent: (event: SSEEvent) => void,
  signal: AbortSignal,
  context: StreamContext = {},
): Promise<void> {
  const delay = createDelay(signal);
  let seq = 1;
  const nextSeq = () => seq++;

  try {
    const flowStep = context.flowStep || 'idle';
    const trimmed = userText.trim();

    if (
      (flowStep === 'awaiting_problem' || flowStep === 'draft_ready') &&
      isCancellation(trimmed)
    ) {
      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'parse',
        label: 'Разбираю запрос',
        status: 'completed',
      });
      await delay(200);

      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'search',
        label: 'Ищу в базе знаний',
        status: 'completed',
      });
      await delay(200);

      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'generate',
        label: 'Формулирую ответ...',
        status: 'in_progress',
      });
      await delay(150);

      const cancelText = 'Создание тикета отменено. Чем ещё я могу помочь?';
      await streamWords(cancelText, onEvent, nextSeq, delay);

      onEvent({
        type: 'flow_step',
        seq: nextSeq(),
        step: 'idle',
      });

      onEvent({
        type: 'done',
        seq: nextSeq(),
        finalText: cancelText,
      });
      return;
    }

    if (flowStep === 'draft_ready' && context.currentDraft && isApproval(trimmed)) {
      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'parse',
        label: 'Проверяю данные черновика',
        status: 'completed',
      });
      await delay(250);

      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'search',
        label: 'Ищу в базе знаний',
        status: 'completed',
      });
      await delay(300);

      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'generate',
        label: 'Создаю тикет в сервисе...',
        status: 'in_progress',
      });
      await delay(400);

      const createdTicket: CreatedTicketPayload = await createTicket({
        subject: context.currentDraft.subject,
        message: context.currentDraft.message,
        mentorId: context.currentDraft.mentor.id,
      });

      const successText = `Отлично! Тикет #${createdTicket.id} «${createdTicket.subject}» успешно отправлен наставнику. Он получит уведомление и ответит вам.`;
      await streamWords(successText, onEvent, nextSeq, delay);

      onEvent({
        type: 'card',
        seq: nextSeq(),
        cardType: 'ticket_success',
        payload: createdTicket,
      });

      onEvent({
        type: 'flow_step',
        seq: nextSeq(),
        step: 'idle',
      });

      onEvent({
        type: 'done',
        seq: nextSeq(),
        finalText: successText,
      });
      return;
    }

    if (flowStep === 'draft_ready' && context.currentDraft) {
      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'parse',
        label: 'Анализирую ваши пожелания',
        status: 'completed',
      });
      await delay(250);

      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'search',
        label: 'Ищу в базе знаний',
        status: 'completed',
      });
      await delay(300);

      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'generate',
        label: 'Обновляю черновик обращения...',
        status: 'in_progress',
      });
      await delay(250);

      const updatedDraft = refineDraft(context.currentDraft, trimmed);

      const refineText =
        'Обновил черновик тикета с учётом ваших правок. Пожалуйста, проверьте — вы можете подтвердить отправку или снова попросить внести изменения:';
      await streamWords(refineText, onEvent, nextSeq, delay);

      onEvent({
        type: 'card',
        seq: nextSeq(),
        cardType: 'ticket_draft',
        payload: updatedDraft,
      });

      onEvent({
        type: 'flow_step',
        seq: nextSeq(),
        step: 'draft_ready',
        draft: updatedDraft,
      });

      onEvent({
        type: 'done',
        seq: nextSeq(),
        finalText: refineText,
      });
      return;
    }

    if (flowStep === 'awaiting_problem') {
      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'parse',
        label: 'Анализирую проблему',
        status: 'completed',
      });
      await delay(300);

      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'search',
        label: 'Ищу в базе знаний',
        status: 'completed',
      });
      await delay(350);

      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'generate',
        label: 'Формирую черновик обращения...',
        status: 'in_progress',
      });
      await delay(250);

      const { subject, message } = extractSubjectAndMessage(trimmed);
      const draftPayload: TicketDraftPayload = {
        id: `draft-${Date.now()}`,
        subject,
        message,
        mentor: DEFAULT_MENTOR,
        status: 'draft',
      };

      const introText =
        'Я подготовил черновик тикета на основе вашего описания. Ознакомьтесь с ним ниже — вы можете подтвердить отправку или написать, что нужно скорректировать:';
      await streamWords(introText, onEvent, nextSeq, delay);

      onEvent({
        type: 'card',
        seq: nextSeq(),
        cardType: 'ticket_draft',
        payload: draftPayload,
      });

      onEvent({
        type: 'flow_step',
        seq: nextSeq(),
        step: 'draft_ready',
        draft: draftPayload,
      });

      onEvent({
        type: 'done',
        seq: nextSeq(),
        finalText: introText,
      });
      return;
    }

    if (isTicketTrigger(trimmed)) {
      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'parse',
        label: 'Разбираю запрос',
        status: 'completed',
      });
      await delay(300);

      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'search',
        label: 'Ищу в базе знаний',
        status: 'completed',
      });
      await delay(350);

      onEvent({
        type: 'stage',
        seq: nextSeq(),
        id: 'generate',
        label: 'Формулирую ответ...',
        status: 'in_progress',
      });
      await delay(200);

      const askProblemText = 'Опиши свою проблему, и я сформирую черновик тикета для наставника.';
      await streamWords(askProblemText, onEvent, nextSeq, delay);

      onEvent({
        type: 'flow_step',
        seq: nextSeq(),
        step: 'awaiting_problem',
      });

      onEvent({
        type: 'done',
        seq: nextSeq(),
        finalText: askProblemText,
      });
      return;
    }

    const isNoAnswer = NO_ANSWER_KEYWORDS.some((kw) => trimmed.toLowerCase().includes(kw));

    onEvent({
      type: 'stage',
      seq: nextSeq(),
      id: 'parse',
      label: 'Разбираю вопрос',
      status: 'completed',
    });
    await delay(300);

    onEvent({
      type: 'stage',
      seq: nextSeq(),
      id: 'search',
      label: 'Ищу в базе знаний',
      status: 'completed',
    });
    await delay(400);

    onEvent({
      type: 'stage',
      seq: nextSeq(),
      id: 'generate',
      label: 'Формулирую ответ...',
      status: 'in_progress',
    });
    await delay(200);

    if (isNoAnswer) {
      await streamWords(NO_ANSWER_TEXT, onEvent, nextSeq, delay);
      await delay(150);

      onEvent({
        type: 'card',
        seq: nextSeq(),
        cardType: 'mentor_fallback',
        payload: DEFAULT_MENTOR,
      });

      onEvent({
        type: 'done',
        seq: nextSeq(),
        finalText: NO_ANSWER_TEXT,
      });
    } else {
      await streamWords(EQUATOR_TEXT, onEvent, nextSeq, delay);
      await delay(150);

      onEvent({
        type: 'card',
        seq: nextSeq(),
        cardType: 'wiki_source',
        payload: {
          title: 'Памятка стажёра: Материалы к экватору',
          section: 'Раздел 2.3 «Промежуточная аттестация»',
          quote:
            '«...заполните шаблон формы обратной связи и согласуйте результаты спринтов с наставником до конца 4-й недели...»',
          url: '/wiki/materials/equator',
        },
      });

      onEvent({
        type: 'done',
        seq: nextSeq(),
        finalText: EQUATOR_TEXT,
      });
    }
  } catch {
    if (signal.aborted) return;
    onEvent({
      type: 'error',
      seq: nextSeq(),
      error: 'Не удалось получить ответ от сервиса. Попробуйте еще раз.',
      canRetry: true,
    });
  }
}
