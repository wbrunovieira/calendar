export class GoogleOAuthToken {
  id: string;
  email: string;
  accessToken: string;
  refreshToken: string;
  expiresAt: Date;
  createdAt: Date;
  updatedAt: Date;

  private constructor(props: {
    id: string;
    email: string;
    accessToken: string;
    refreshToken: string;
    expiresAt: Date;
    createdAt: Date;
    updatedAt: Date;
  }) {
    Object.assign(this, props);
  }

  static create(data: {
    id?: string;
    email: string;
    accessToken: string;
    refreshToken: string;
    expiresAt: Date;
    createdAt?: Date;
    updatedAt?: Date;
  }): GoogleOAuthToken {
    return new GoogleOAuthToken({
      id: data.id || crypto.randomUUID(),
      email: data.email,
      accessToken: data.accessToken,
      refreshToken: data.refreshToken,
      expiresAt: data.expiresAt,
      createdAt: data.createdAt || new Date(),
      updatedAt: data.updatedAt || new Date(),
    });
  }

  isExpired(bufferMs = 300_000): boolean {
    return this.expiresAt.getTime() - Date.now() < bufferMs;
  }
}
