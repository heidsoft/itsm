import { httpClient } from '@/lib/api/http-client';

export interface Application {
  id: number;
  name: string;
  code: string;
  type?: string;
  status?: string;
  project_id?: number;
  created_at?: string;
  updated_at?: string;
  edges?: {
    microservices?: Microservice[];
  };
}

export interface Microservice {
  id: number;
  name: string;
  code: string;
  language?: string;
  framework?: string;
  status?: string;
  application_id?: number;
}

export interface CreateApplicationRequest {
  name: string;
  code: string;
  type?: string;
  project_id?: number;
}

export interface CreateMicroserviceRequest {
  name: string;
  code: string;
  language?: string;
  framework?: string;
  application_id: number;
}

class ApplicationService {
  private readonly baseUrl = '/api/v1/applications';

  async listApplications(): Promise<Application[]> {
    return httpClient.get<Application[]>(this.baseUrl);
  }

  async createApplication(data: CreateApplicationRequest): Promise<Application> {
    return httpClient.post<Application>(this.baseUrl, data);
  }

  async createMicroservice(data: CreateMicroserviceRequest): Promise<Microservice> {
    return httpClient.post<Microservice>(`${this.baseUrl}/microservices`, data);
  }

  async updateApplication(id: number, data: Partial<CreateApplicationRequest>): Promise<Application> {
    return httpClient.put<Application>(`${this.baseUrl}/${id}`, data);
  }

  async deleteApplication(id: number): Promise<void> {
    return httpClient.delete(`${this.baseUrl}/${id}`);
  }

  async listMicroservices(): Promise<Microservice[]> {
    return httpClient.get<Microservice[]>(`${this.baseUrl}/microservices`);
  }

  async updateMicroservice(id: number, data: Partial<CreateMicroserviceRequest>): Promise<Microservice> {
    return httpClient.put<Microservice>(`${this.baseUrl}/microservices/${id}`, data);
  }

  async deleteMicroservice(id: number): Promise<void> {
    return httpClient.delete(`${this.baseUrl}/microservices/${id}`);
  }
}

export const applicationService = new ApplicationService();

