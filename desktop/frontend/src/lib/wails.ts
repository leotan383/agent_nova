export interface NovelCard {
  id: string;
  path: string;
  title: string;
  genre: string;
  phase: string;
  current_volume: number;
  current_chapter: number;
  chapter_count: number;
  written_words?: number;
  target_words?: number;
  progress_percent?: number;
  pinned: boolean;
  archived: boolean;
  last_opened_at: string;
  missing: boolean;
}

export interface StatusReport {
  phase: string;
  title: string;
  current_volume: number;
  current_chapter: number;
  chapter_count: number;
  open_foreshadows: number;
  memory_count: number;
  written_words: number;
  target_words: number;
  chapter_words_goal: number;
  progress_percent: number;
  remaining_words: number;
  estimated_total_chapters: number;
  remaining_chapters: number;
  avg_words_per_chapter: number;
  style?: string;
  urgent?: string[];
  next_steps?: string[];
}

export interface ChapterDTO {
  number: number;
  title: string;
  word_count: number;
  status: string;
}

export interface CreateNovelInput {
  dir: string;
  title: string;
  genre: string;
  style: string;
  target_words: number;
  chapter_words: number;
  synopsis: string;
  tone?: string;
  protagonist?: string;
  cheat?: string;
}

export const genreOptions = ["玄幻", "都市", "科幻", "仙侠", "历史", "游戏", "悬疑", "其他"] as const;

export const styleOptions = [
  "热血",
  "爽文",
  "黑暗",
  "轻松",
  "轻松搞笑",
  "慢热",
  "群像",
  "无敌流",
  "系统流",
  "迪化流",
] as const;

export const targetWordOptions = [
  { value: 100000, label: "10 万字" },
  { value: 200000, label: "20 万字" },
  { value: 300000, label: "30 万字" },
  { value: 500000, label: "50 万字" },
  { value: 1000000, label: "100 万字" },
] as const;

export const chapterWordOptions = [
  { value: 3000, label: "3000 字/章" },
  { value: 4000, label: "4000 字/章" },
  { value: 5000, label: "5000 字/章" },
] as const;

export interface StartWriteInput {
  chapter: number;
  volume: number;
  resume: boolean;
}

export interface WriteJobInfo {
  id: string;
  chapter: number;
  volume: number;
  status: string;
}

export interface WriteJobStateDTO {
  stream_text: string;
  step: string;
  step_message: string;
}

export interface TokenUsageDTO {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  estimated_cost_usd?: number;
}

export interface ProjectTokenUsageDTO {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  write_runs: number;
  estimated_cost_usd?: number;
}

export interface WriteReportDTO {
  stage: string;
  status: string;
  summary: string;
  artifacts?: string[];
  issues?: string[];
  next_steps?: string[];
  token_usage?: TokenUsageDTO;
}

export interface TodoItemDTO {
  id: string;
  label: string;
  detail: string;
  severity: string;
  action: string;
  action_param?: string;
}

export interface ProjectHealthDTO {
  ok: boolean;
  suggested_volume: number;
  next_chapter: number;
  has_volume_outline: boolean;
  volume_outline_path?: string;
  todos: TodoItemDTO[];
}

export interface StartPlanInput {
  volume: number;
}

export interface PlanJobInfo {
  id: string;
  volume: number;
  status: string;
}

export interface PlanReportDTO {
  stage: string;
  status: string;
  summary: string;
  artifacts?: string[];
  next_steps?: string[];
}

export interface VolumeOutlineDTO {
  volume: number;
  path: string;
  body: string;
  exists: boolean;
}

export interface StartReviewInput {
  chapter: number;
}

export interface ReviewJobInfo {
  id: string;
  chapter: number;
  status: string;
}

export interface ReviewReportDTO {
  stage: string;
  status: string;
  summary: string;
  artifacts?: string[];
  issues?: string[];
  next_steps?: string[];
}

export interface ChapterReviewMetricsDTO {
  chapter: number;
  exists: boolean;
  hook_score: number;
  cool_point: string;
  debt: string;
  issues: string[];
}

export interface ActiveWriteJobDTO {
  active: boolean;
  job: WriteJobInfo;
  state: WriteJobStateDTO;
}

export interface ActivePlanJobDTO {
  active: boolean;
  job: PlanJobInfo;
}

export interface ActiveReviewJobDTO {
  active: boolean;
  job: ReviewJobInfo;
}

export interface MemoryConflictDTO {
  subject: string;
  count: number;
  memories: MemoryDTO[];
}

export interface MemoryDTO {
  id: string;
  category: string;
  subject: string;
  content: string;
  source_chapter: number;
  status: string;
}

export interface ForeshadowDTO {
  id: string;
  description: string;
  planted_chapter: number;
  resolved_chapter: number;
  status: string;
}

export interface CoachTurnDTO {
  role: string;
  content: string;
}

export interface ReviseJobInfo {
  id: string;
  chapter: number;
  status: string;
}

export interface SelectionTransformInput {
  chapter: number;
  action: "polish" | "expand" | "shorten" | "dialogue" | "custom";
  selected_text: string;
  custom_prompt?: string;
}

export interface SelectionJobInfo {
  id: string;
  chapter: number;
  action: string;
  status: string;
}

export interface GateCheckDTO {
  key: string;
  label: string;
  ok: boolean;
  detail: string;
  blocking: boolean;
}

export interface WriteGateDTO {
  ok: boolean;
  chapter: number;
  volume: number;
  checks: GateCheckDTO[];
}

export interface EntityDTO {
  id: string;
  type: string;
  name: string;
  state: Record<string, string>;
  last_chapter: number;
}

export interface AppConfigDTO {
  model: string;
  base_url: string;
  has_api_key: boolean;
  api_key_mask: string;
}

export interface SaveAppConfigInput {
  model: string;
  base_url: string;
  api_key?: string;
}

export interface DiscoverPreviewDTO {
  title: string;
  genre: string;
  style: string;
  protagonist: string;
  cheat: string;
  pitch: string;
  synopsis: string;
  transcript: string;
}

export interface CreateNovelFromDiscoverInput {
  dir: string;
  title: string;
  genre: string;
  style: string;
  target_words: number;
  chapter_words: number;
  protagonist: string;
  cheat: string;
  synopsis: string;
  enrich: boolean;
}

export interface ExportInput {
  format: string;
  out_path: string;
  from_chapter: number;
  to_chapter: number;
}

export interface ExportResultDTO {
  path: string;
  format: string;
  chapter_count: number;
  word_count: number;
}

export interface VersionEntryDTO {
  id: string;
  created_at: string;
  source: string;
  label: string;
  word_count: number;
  file: string;
}

export interface DiffLineDTO {
  type: "add" | "del" | "same" | string;
  text: string;
}

export interface DiffResultDTO {
  from_id: string;
  to_id: string;
  from_label: string;
  to_label: string;
  lines: DiffLineDTO[];
  added_words: number;
  removed_words: number;
}

export interface WikiEntryDTO {
  id: string;
  group: string;
  title: string;
  subtitle: string;
  kind: string;
  path?: string;
}

export interface WikiContentDTO {
  id: string;
  title: string;
  group: string;
  kind: string;
  body: string;
  path?: string;
  can_open: boolean;
  editable: boolean;
}

export interface ChapterDocDTO {
  kind: string;
  chapter: number;
  title: string;
  body: string;
  exists: boolean;
  path?: string;
}

export interface SearchHitDTO {
  kind: string;
  id: string;
  title: string;
  snippet: string;
  chapter: number;
  wiki_id?: string;
}

export interface WriteContextDTO {
  chapter: number;
  volume: number;
  outline: string;
  recent_summary: string;
  settings: string;
  memories: string;
  fts_hits: string;
  open_foreshadows: string;
}

export interface UpdateMemoryInput {
  id: string;
  category: string;
  subject: string;
  content: string;
  source_chapter: number;
}

export interface CreateMemoryInput {
  category: string;
  subject: string;
  content: string;
  source_chapter: number;
}

interface AppBindings {
  ListNovels(includeArchived: boolean): Promise<NovelCard[]>;
  GetActiveNovel(): Promise<{ id?: string; path?: string }>;
  SwitchNovel(id: string): Promise<void>;
  RegisterNovel(path: string): Promise<NovelCard>;
  PickNovelDirectory(): Promise<string>;
  RemoveFromLibrary(id: string): Promise<void>;
  SetNovelArchived(id: string, archived: boolean): Promise<void>;
  SetNovelPinned(id: string, pinned: boolean): Promise<void>;
  CreateNovel(input: CreateNovelInput): Promise<NovelCard>;
  PickCreateDirectory(): Promise<string>;
  RevealInFolder(path: string): Promise<void>;
  GetStatus(): Promise<StatusReport>;
  GetProjectHealth(): Promise<ProjectHealthDTO>;
  ListChapters(): Promise<ChapterDTO[]>;
  GetChapterContent(number: number): Promise<string>;
  SaveChapterContent(number: number, content: string): Promise<void>;
  GetChapterDocument(chapter: number, kind: string): Promise<ChapterDocDTO>;
  SaveChapterDocument(chapter: number, kind: string, body: string): Promise<void>;
  SearchProject(query: string, limit: number): Promise<SearchHitDTO[]>;
  GetWriteContext(chapter: number, volume: number): Promise<WriteContextDTO>;
  GetWriteGate(chapter: number, volume: number): Promise<WriteGateDTO>;
  ListEntities(entityType: string): Promise<EntityDTO[]>;
  GetAppConfig(): Promise<AppConfigDTO>;
  SaveAppConfig(input: SaveAppConfigInput): Promise<void>;
  StartDiscover(seedGenre: string): Promise<CoachTurnDTO[]>;
  SendDiscoverMessage(message: string): Promise<void>;
  FinishDiscover(): Promise<DiscoverPreviewDTO>;
  CreateNovelFromDiscover(input: CreateNovelFromDiscoverInput): Promise<NovelCard>;
  GetDiscoverTurns(): Promise<CoachTurnDTO[] | null>;
  ClearDiscover(): Promise<void>;
  DefaultExportFilename(format: string): Promise<string>;
  PickExportPath(format: string, defaultName: string): Promise<string>;
  ExportProject(input: ExportInput): Promise<ExportResultDTO>;
  UpdateMemory(input: UpdateMemoryInput): Promise<void>;
  ArchiveMemory(id: string): Promise<void>;
  CreateMemory(input: CreateMemoryInput): Promise<MemoryDTO>;
  ResolveForeshadow(id: string, resolvedChapter: number): Promise<void>;
  UpdateForeshadow(id: string, description: string): Promise<void>;
  FindMemoryConflicts(): Promise<MemoryConflictDTO[]>;
  HasAPIKey(): Promise<boolean>;
  AppInfo(): Promise<{ name: string; version: string }>;
  StartWriteChapter(input: StartWriteInput): Promise<WriteJobInfo>;
  CancelWriteChapter(jobID: string): Promise<void>;
  GetWriteJob(jobID: string): Promise<WriteJobInfo>;
  GetWriteJobState(jobID: string): Promise<WriteJobStateDTO>;
  IsWriteRunning(): Promise<boolean>;
  GetActiveWriteJob(): Promise<ActiveWriteJobDTO>;
  GetProjectTokenUsage(): Promise<ProjectTokenUsageDTO>;
  GetVolumeOutline(volume: number): Promise<VolumeOutlineDTO>;
  SaveVolumeOutline(volume: number, body: string): Promise<void>;
  StartPlanVolume(input: StartPlanInput): Promise<PlanJobInfo>;
  CancelPlanVolume(jobID: string): Promise<void>;
  IsPlanRunning(): Promise<boolean>;
  GetActivePlanJob(): Promise<ActivePlanJobDTO>;
  RebuildProjectIndex(): Promise<void>;
  StartReviewChapter(input: StartReviewInput): Promise<ReviewJobInfo>;
  CancelReviewChapter(jobID: string): Promise<void>;
  IsReviewRunning(): Promise<boolean>;
  GetActiveReviewJob(): Promise<ActiveReviewJobDTO>;
  GetChapterReviewMetrics(chapter: number): Promise<ChapterReviewMetricsDTO>;
  ListMemories(): Promise<MemoryDTO[]>;
  ListForeshadows(status: string): Promise<ForeshadowDTO[]>;
  SendChapterCoachMessage(chapter: number, message: string): Promise<void>;
  GetChapterCoachTurns(chapter: number): Promise<CoachTurnDTO[] | null>;
  ClearChapterCoach(chapter: number): Promise<void>;
  StartChapterRevision(chapter: number): Promise<ReviseJobInfo>;
  CancelChapterRevision(jobID: string): Promise<void>;
  StartSelectionTransform(input: SelectionTransformInput): Promise<SelectionJobInfo>;
  CancelSelectionTransform(jobID: string): Promise<void>;
  ApplyChapterContent(chapter: number, content: string): Promise<void>;
  ListChapterVersions(chapter: number): Promise<VersionEntryDTO[]>;
  PreviewChapterDiff(chapter: number, newContent: string): Promise<DiffResultDTO>;
  DiffChapterVersions(chapter: number, fromID: string, toID: string): Promise<DiffResultDTO>;
  RestoreChapterVersion(chapter: number, versionID: string): Promise<void>;
  ListWikiEntries(): Promise<WikiEntryDTO[]>;
  GetWikiContent(id: string): Promise<WikiContentDTO>;
  SaveWikiContent(id: string, body: string): Promise<void>;
}

declare global {
  interface Window {
    go?: {
      main?: {
        App?: AppBindings;
      };
    };
  }
}

export function app(): AppBindings {
  const bindings = window.go?.main?.App;
  if (!bindings) {
    throw new Error("Wails bindings unavailable — run via wails dev or Nova Studio app");
  }
  return bindings;
}

export const phaseLabel: Record<string, string> = {
  empty: "空",
  init_done: "已立项",
  planning: "规划中",
  writing: "连载中",
  paused: "已暂停",
};

export function formatRelativeTime(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  const diff = Date.now() - d.getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "刚刚";
  if (mins < 60) return `${mins} 分钟前`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours} 小时前`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days} 天前`;
  return d.toLocaleDateString("zh-CN");
}

/** 字数友好展示，如 3.2 万字 */
export function formatWordCount(n: number): string {
  if (n >= 10000) {
    const wan = n / 10000;
    return Number.isInteger(wan) ? `${wan} 万字` : `${wan.toFixed(1)} 万字`;
  }
  return `${n.toLocaleString("zh-CN")} 字`;
}
