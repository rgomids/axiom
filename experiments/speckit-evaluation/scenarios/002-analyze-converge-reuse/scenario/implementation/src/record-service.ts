// Frozen, non-executable experiment fixture. Not Axiom product code.

export type RecordInput = { payload: string };
export type StoredRecord = { id: string; payload: string };

export interface IdGenerator {
  generate(): string;
}

export interface PostgresRecordRepository {
  insert(record: StoredRecord): Promise<void>;
  find(id: string): Promise<StoredRecord | undefined>;
}

export interface RedisRecordCache {
  put(record: StoredRecord): Promise<void>;
  find(id: string): Promise<StoredRecord | undefined>;
}

export class RecordService {
  constructor(
    private readonly ids: IdGenerator,
    private readonly repository: PostgresRecordRepository,
    private readonly cache: RedisRecordCache,
  ) {}

  async create(input: RecordInput): Promise<StoredRecord> {
    const record = { id: this.ids.generate(), payload: input.payload };
    await this.repository.insert(record);
    await this.cache.put(record);
    return record;
  }

  async retrieve(id: string): Promise<StoredRecord | undefined> {
    const cached = await this.cache.find(id);
    if (cached) return cached;

    const stored = await this.repository.find(id);
    if (stored) await this.cache.put(stored);
    return stored;
  }
}
