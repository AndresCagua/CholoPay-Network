import { bootstrapApplication } from "@angular/platform-browser";
import { AppComponent } from "./app/app.component";

bootstrapApplication(AppComponent).catch((error: unknown) => {
  console.error("Falló el inicio de la aplicación de Banco Sierra", error);
});
