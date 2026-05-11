import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { AuthService } from '../services/auth.service';
import { kycGuard } from './kyc.guard';
import { signal } from '@angular/core';

describe('kycGuard', () => {
  let authService: jasmine.SpyObj<AuthService>;
  let router: Router;

  beforeEach(() => {
    const authSpy = jasmine.createSpyObj('AuthService', ['isKycBlocked'], {
      isKycBlocked: signal(false),
    });

    TestBed.configureTestingModule({
      providers: [
        provideRouter([
          { path: 'profile', component: {} as any },
          { path: 'dashboard', canActivate: [kycGuard], component: {} as any },
        ]),
        { provide: AuthService, useValue: authSpy },
      ],
    });

    authService = TestBed.inject(AuthService) as jasmine.SpyObj<AuthService>;
    router = TestBed.inject(Router);
  });

  it('should allow navigation when KYC is not blocked', () => {
    // isKycBlocked returns false by default from our spy
    const result = TestBed.runInInjectionContext(() => kycGuard({} as any, {} as any));
    expect(result).toBeTrue();
  });

  it('should describe the guard purpose', () => {
    // Verify the guard is exported and callable
    expect(kycGuard).toBeDefined();
    expect(typeof kycGuard).toBe('function');
  });
});
